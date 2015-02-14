// file system utility

package daemon

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"syscall"

	"code.google.com/p/go.exp/fsnotify"
	"github.com/andres-erbsen/chatterbox/client/encoding"
	"github.com/andres-erbsen/chatterbox/client/persistence"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	"github.com/andres-erbsen/chatterbox/shred"
	dename "github.com/andres-erbsen/dename/protocol"
)

func (d *Daemon) keysDir() string        { return filepath.Join(d.RootDir, "keys") }
func (d *Daemon) profilesDir() string    { return filepath.Join(d.RootDir, "profile") }
func (d *Daemon) prekeysPath() string    { return filepath.Join(d.keysDir(), "prekeys.pb") }
func (d *Daemon) ratchetKeysDir() string { return filepath.Join(d.keysDir(), "ratchet") }
func (d *Daemon) configPath() string     { return filepath.Join(d.keysDir(), "config.pb") }

func (d *Daemon) ourChatterboxProfilePath() string {
	return filepath.Join(d.keysDir(), "chatterbox-profile.pb")
}

func (d *Daemon) ratchetPath(name string) string {
	return filepath.Join(d.ratchetKeysDir(), encoding.EscapeFilename(name))
}
func (d *Daemon) profilePath(name string) string {
	return filepath.Join(d.profilesDir(), encoding.EscapeFilename(name))
}

// Copy copyes the contents of file source to dest. NOT atomic.
func Copy(source string, dest string, perm os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)

	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)

	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}

func LoadPrekeys(d *Daemon) ([]*[32]byte, []*[32]byte, error) {
	prekeysProto := new(proto.Prekeys)
	err := persistence.UnmarshalFromFile(d.prekeysPath(), prekeysProto)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	if len(prekeysProto.PrekeyPublics) != len(prekeysProto.PrekeySecrets) {
		return nil, nil, fmt.Errorf("len(prekeysProto.prekeyPublics) != len(prekeysProto.prekeySecrets)")
	}
	// convert protobuf proto.Byte32 to *[32]byte
	prekeySecrets := make([]*[32]byte, len(prekeysProto.PrekeySecrets))
	prekeyPublics := make([]*[32]byte, len(prekeysProto.PrekeyPublics))
	for i := 0; i < len(prekeySecrets); i++ {
		prekeySecrets[i] = (*[32]byte)(&prekeysProto.PrekeySecrets[i])
		prekeyPublics[i] = (*[32]byte)(&prekeysProto.PrekeyPublics[i])
	}
	return prekeyPublics, prekeySecrets, nil
}

func StorePrekeys(d *Daemon, prekeyPublics, prekeySecrets []*[32]byte) error {
	if len(prekeyPublics) != len(prekeySecrets) {
		panic("len(prekeysPublics) != len(prekeySecrets)")
	}
	// convert [32]byte to proto.Byte32
	prekeysProto := proto.Prekeys{
		PrekeySecrets: make([]proto.Byte32, len(prekeySecrets)),
		PrekeyPublics: make([]proto.Byte32, len(prekeySecrets)),
	}
	for i := 0; i < len(prekeyPublics); i++ {
		prekeysProto.PrekeySecrets[i] = (proto.Byte32)(*prekeySecrets[i])
		prekeysProto.PrekeyPublics[i] = (proto.Byte32)(*prekeyPublics[i])
	}
	return d.MarshalToFile(d.prekeysPath(), &prekeysProto)
}

func StoreLocalAccountConfig(d *Daemon, localAccountConfig *proto.LocalAccountConfig) error {
	return d.MarshalToFile(d.configPath(), localAccountConfig)
}

func LoadRatchet(d *Daemon, name string, fillAuth func(tag, data []byte, theirAuthPublic *[32]byte), checkAuth func(tag, data, msg []byte, ourAuthPrivate *[32]byte) error) (*ratchet.Ratchet, error) {
	ratch := new(ratchet.Ratchet)
	if err := persistence.UnmarshalFromFile(d.ratchetPath(name), ratch); err != nil {
		return nil, err
	}
	ratch.FillAuth = fillAuth
	ratch.CheckAuth = checkAuth
	return ratch, nil
}

func StoreRatchet(d *Daemon, name string, ratch *ratchet.Ratchet) error {
	return d.MarshalToFile(d.ratchetPath(name), ratch)
}

func (d *Daemon) LatestProfile(name string, received *dename.Profile) (*dename.Profile, error) {
	stored := new(dename.Profile)
	err := persistence.UnmarshalFromFile(d.profilePath(name), stored)
	if err != nil || *received.Version > *stored.Version {
		return received, d.MarshalToFile(d.profilePath(name), received)
	}
	return stored, nil
}

func AllRatchets(d *Daemon, fillAuth func(tag, data []byte, theirAuthPublic *[32]byte), checkAuth func(tag, data, msg []byte, ourAuthPrivate *[32]byte) error) ([]*ratchet.Ratchet, error) {
	files, err := ioutil.ReadDir(d.ratchetKeysDir())
	if err != nil {
		return nil, err
	}
	ret := make([]*ratchet.Ratchet, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ratch := new(ratchet.Ratchet)
		err := persistence.UnmarshalFromFile(filepath.Join(d.ratchetKeysDir(), file.Name()), ratch)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ratchet for \"%s\": %s", file.Name(), err)
		}
		ratch.FillAuth = fillAuth
		ratch.CheckAuth = checkAuth
		ret = append(ret, ratch)
	}
	return ret, nil
}

func InitFs(d *Daemon) error {
	// create root directory and immediate sub directories
	os.MkdirAll(d.RootDir, 0700)
	subdirs := []string{
		d.ConversationDir(),
		d.OutboxDir(),
		d.TempDir(),
		d.keysDir(),
		d.profilesDir(),
		d.ratchetKeysDir(),
	}
	for _, dir := range subdirs {
		os.MkdirAll(dir, 0700) // FIXME: handle error
	}

	// for each existing conversation, create a folder in the outbox
	copyToOutbox := func(cPath string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if cPath != d.ConversationDir() {
			if f.IsDir() {
				// create the outbox directory in tmp, then (atomically) move it to outbox
				tmpDir, err := d.MkdirInTemp()
				if err != nil {
					return err
				}
				defer shred.RemoveAll(tmpDir)
				conversationInfo, err := os.Stat(cPath)
				if err != nil {
					return err
				}
				var c_perm = conversationInfo.Mode()
				metadataFile := filepath.Join(cPath, persistence.MetadataFileName)
				metadataInfo, err := os.Stat(metadataFile)
				if err != nil {
					return err
				}
				var m_perm = metadataInfo.Mode()
				oldUmask := syscall.Umask(0000)
				defer syscall.Umask(oldUmask)
				os.Mkdir(filepath.Join(tmpDir, path.Base(cPath)), c_perm)
				err = Copy(metadataFile, filepath.Join(tmpDir, path.Base(cPath), persistence.MetadataFileName), m_perm)
				if err != nil {
					return err
				}
				err = os.Rename(filepath.Join(tmpDir, path.Base(cPath)), filepath.Join(d.OutboxDir(), path.Base(cPath)))
				if err != nil {
					// skip this conversation; this probably means it already exists in the outbox
					return nil
				}
			}
		}
		return nil
	}
	err := filepath.Walk(d.ConversationDir(), copyToOutbox)
	if err != nil {
		return err
	}
	return nil
}

func WatchDir(watcher *fsnotify.Watcher, dir string, initFn filepath.WalkFunc) error {
	registerAndInit := func(path string, f os.FileInfo, err error) error {
		err = watcher.Watch(path)
		if err != nil {
			return err
		}
		err = initFn(path, f, err)
		if err != nil {
			return err
		}
		return nil
	}

	err := filepath.Walk(dir, registerAndInit)
	if err != nil {
		return err
	}

	return nil
}
