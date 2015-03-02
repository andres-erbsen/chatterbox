import QtQuick 2.2
import QtQuick.Controls 1.1
import QtQuick.Layouts 1.1


ApplicationWindow {
	id: newConversationWindow
    visible: true
    title: "New Conversation"
    property int margin: 5
    width: mainLayout.implicitWidth + 2 * margin
    height: mainLayout.implicitHeight + 2 * margin
    minimumWidth: mainLayout.Layout.minimumWidth + 40 * margin
    minimumHeight: mainLayout.Layout.minimumHeight + 12 * margin

    function closeWindow() {
    	newConversationWindow.close();
    }

	Action {
		id: sendMessage
		objectName: "sendMessage"
		text: "Send &Message"
		shortcut: "Ctrl+Return"
	}

    ColumnLayout {
        id: mainLayout
        anchors.fill: parent
        anchors.margins: margin
		RowLayout {
			Text {text: "To:"}
				TextField {
					id: toField
					objectName: "toField"
					focus: true
					placeholderText: "dename names, comma-separated"
					Layout.fillWidth: true
					onAccepted: {subjectField.focus = true}
				}
		}

		RowLayout {
			Text {text: "Subject:"}
				TextField {
					id: subjectField
					objectName: "subjectField"
					Layout.fillWidth: true
					onAccepted: {messageArea.focus = true}
				}
		}


		TextArea {
			id: messageArea 
			objectName: "messageArea"
			text: "Ctrl + Enter to send a message."
			Layout.minimumHeight: 10
			Layout.fillWidth: true
			Layout.fillHeight: true
			textFormat: TextEdit.PlainText
			wrapMode: TextEdit.Wrap
			Component.onCompleted: {
				messageArea.selectAll()
			}
		}
    }
}
