import QtQuick 2.2
import QtQuick.Controls 1.1
import QtQuick.Layouts 1.1


ApplicationWindow {
	id: conversationWindow
	signal sendMessage(string message)

    visible: true
    title: "Conversation"
    property int margin: 10
    width: mainLayout.implicitWidth + 2 * margin
    height: mainLayout.implicitHeight + 2 * margin
    minimumWidth: mainLayout.Layout.minimumWidth + 40 * margin
    minimumHeight: mainLayout.Layout.minimumHeight + 12 * margin

	Action {
		id: sendMessage
		text: "Send &Message"
		shortcut: "Ctrl+Return"
		onTriggered: {
			conversationWindow.sendMessage(messageArea.text);
			messageArea.remove(0, messageArea.length);
		}
	}

	ListModel {
		id: messageModel
		objectName: 'messageModel'

		function addItem(json) {
			console.log(json)
			var parsed = JSON.parse(json);
			for (var key in parsed) {
				if (parsed.hasOwnProperty(key) && (typeof parsed[key] == 'object')) {
						//console.log(key);
						parsed[key] = parsed[key].toString();
				}
			}
			parsed['objectName'] = parsed['Subject'];
			append(parsed);
		}
    }

    ColumnLayout {
        id: mainLayout
        anchors.fill: parent
        anchors.margins: margin
        spacing: 20
		RowLayout {
			anchors.top: parent.top
			ListView {
				id: messageView
		        objectName: "messageView"

		        anchors.fill: parent
		        model: messageModel
		        delegate: RowLayout {
		        	Text{ 
		        		anchors.top: parent.top
		        		text: Sender + ": "
		        		textFormat: Text.PlainText
		        		font.bold:true
		        	}
		        	Text{ 
		        		anchors.top: parent.top
		        		text: Content
		        		textFormat: Text.PlainText
		        	}

		        }

		        Layout.minimumWidth: 100
		        Layout.minimumHeight: 100
		        Layout.preferredWidth: 300
		        Layout.preferredHeight: 180
			}
		}


		TextArea {
			anchors.bottom: parent.bottom
			id: messageArea 
			objectName: "messageArea"
			text: "Ctrl + Enter to send a message."
			Layout.fillWidth: true
			Layout.minimumHeight: 12
			Layout.preferredHeight: 36
			textFormat: TextEdit.PlainText
			wrapMode: TextEdit.Wrap
			
		}
    }
}
