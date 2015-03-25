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

	SplitView {
        id: mainLayout
		anchors.fill: parent
		orientation: Qt.Vertical

		TextArea {
			id: historyArea
			objectName: "historyArea"
			Layout.fillHeight: true

			readOnly: true
			wrapMode: TextEdit.Wrap
			textFormat: TextEdit.RichText
			verticalAlignment: TextEdit.AlignTop
		}

		TextArea {
			id: inputArea 
			objectName: "inputArea"
			Layout.minimumHeight: 18
			Layout.preferredHeight: 36

			text: "Ctrl + Enter to send a message."
			textFormat: TextEdit.PlainText
			wrapMode: TextEdit.Wrap

			focus: true
			Component.onCompleted: {
				inputArea.selectAll()
				inputArea.height = 36;
			}
		}
    }
}
