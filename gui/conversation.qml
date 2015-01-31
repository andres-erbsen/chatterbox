import QtQuick 2.2
import QtQuick.Controls 1.1
import QtQuick.Layouts 1.1

ApplicationWindow {
    visible: true
    title: "New Conversation"
    property int margin: 6
    width: mainLayout.implicitWidth + 2 * margin
    height: mainLayout.implicitHeight + 2 * margin
    minimumWidth: mainLayout.Layout.minimumWidth + 40 * margin
    minimumHeight: mainLayout.Layout.minimumHeight + 12 * margin

    ColumnLayout {
        id: mainLayout
        anchors.fill: parent
        anchors.margins: margin
		RowLayout {
			Text {text: "To:"}
				TextField {
					id: toField
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
					Layout.fillWidth: true
					onAccepted: {messageArea.selectAll(); messageArea.focus = true}
				}
		}


		TextArea {
			id: messageArea 
			text: "Write the message here and press Ctrl+Return to send"
			Layout.minimumHeight: 10
			Layout.fillWidth: true
			Layout.fillHeight: true
			textFormat: TextEdit.PlainText
			wrapMode: TextEdit.Wrap
			/* andreser: the following works for me
			keys.onreturnpressed: {
				console.log("return pressed in main textarea");
			}
			*/
		}

		Action {
			id: sendMessage
			text: "Send &Message"
			shortcut: "Ctrl+Return"
			onTriggered: {
				console.log("sendMessage")
			}
		}

		/*
        GroupBox {
            id: sendMessageButton
            Layout.alignment: Qt.AlignRight
            flat: true
			Button {
				text: "Send Message"
				onClicked: {sendMessage.trigger()}
			}
        }
		*/
    }
}
