import QtQuick 2.2
import QtQuick.Controls 1.1
import QtQuick.Layouts 1.1


ApplicationWindow {
	id: conversationWindow
    visible: true
    title: "Chatterbox Conversation"

	signal sendMessage(string message)
	Action {
		id: sendMessage
		text: "Send &Message"
		shortcut: "Ctrl+Return"
		onTriggered: {
			conversationWindow.sendMessage(inputArea.text);
			inputArea.remove(0, inputArea.length);
		}
	}

    ColumnLayout {
        id: mainLayout
        anchors.fill: parent

		TextArea {
			anchors {
				bottom: inputArea.top
				top: parent.top
			}
			Layout.fillWidth: true
			id: historyArea
			objectName: "historyArea"
			readOnly: true
			wrapMode: TextEdit.Wrap
			textFormat: TextEdit.RichText
			verticalAlignment: TextEdit.AlignTop
		}

		TextArea {
			id: inputArea 
			objectName: "inputArea"
			text: "Ctrl + Enter to send a message."
			anchors.bottom: parent.bottom
			Layout.fillWidth: true
			Layout.minimumHeight: 12
			Layout.preferredHeight: 42
			textFormat: TextEdit.PlainText
			wrapMode: TextEdit.Wrap

			focus: true
			Component.onCompleted: {
				inputArea.selectAll()
			}
		}
    }
}
