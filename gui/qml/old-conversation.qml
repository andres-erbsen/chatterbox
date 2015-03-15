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

		function addItem(json) { append(JSON.parse(json)); }
    }

    ColumnLayout {
        id: mainLayout
        anchors.fill: parent
        anchors.margins: margin

		ScrollView {
			// TODO: handle pageup, pagedown
			Layout.fillHeight: true
			Layout.fillWidth: true
			ListView {
				id: messageView
				objectName: "messageView"

				model: messageModel
				delegate: RowLayout {
						Text{ 
							id: sender_text
							anchors.top: parent.top
							text: Sender + ": "
							textFormat: Text.PlainText
							font.bold:true
						}
						Text{ 
							id:content_text
							Layout.maximumWidth:messageView.width - sender_text.width
							Layout.preferredWidth:messageView.width - sender_text.width
							text: Content
							textFormat: Text.PlainText
							wrapMode: Text.Wrap
						}

				}
			}
		}

		TextArea {
			id: messageArea 
			objectName: "messageArea"
			text: "Ctrl + Enter to send a message."
			Layout.fillWidth: true
			Layout.minimumHeight: 12
			Layout.preferredHeight: 36
			textFormat: TextEdit.PlainText
			wrapMode: TextEdit.Wrap

			focus: true
			Component.onCompleted: {
				messageArea.selectAll()
			}
		}
    }
}
