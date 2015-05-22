1. Install `go`, `TOR`, [`dename`](https://github.com/andres-erbsen/dename) and [get an account](https://dename.mit.edu/).

2. Download, compile, install

		go get -u github.com/andres-erbsen/chatterbox/{chatterboxd,chatterbox-init,chatterbox-create,chatterbox-qt}

3. Create an account:

		chatterbox-init  -dename=${DENAME_USER}

4. Start the daemon

        chatterboxd ${INIT_DIR}

5. Run the qt UI

		# Install QT packages -- these are the packages for Arch
        pacman -S qt5-base qt5-connectivity qt5-declarative qt5-enginio qt5-graphicaleffects qt5-imageformats qt5-location qt5-multimedia qt5-quick1 qt5-quickcontrols qt5-script qt5-sensors qt5-serialport qt5-svg qt5-tools qt5-translations qt5-wayland qt5-webchannel qt5-webengine qt5-webkit qt5-websockets qt5-x11extras qt5-xmlpatterns

		chatterbox-qt -root=${INIT_DIR}
