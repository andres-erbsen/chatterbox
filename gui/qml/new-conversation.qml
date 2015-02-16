import QtQuick 2.2
import QtQuick.Controls 1.1
import QtQuick.Layouts 1.1

ApplicationWindow {
	id: newConversationWindow
	signal sendMessage(string to, string subject, string message)

    visible: true
    title: "New Conversation"
    property int margin: 5
    width: mainLayout.implicitWidth + 2 * margin
    height: mainLayout.implicitHeight + 2 * margin
    minimumWidth: mainLayout.Layout.minimumWidth + 40 * margin
    minimumHeight: mainLayout.Layout.minimumHeight + 12 * margin

	Action {
		id: sendMessage
		text: "Send &Message"
		shortcut: "Ctrl+Return"
		onTriggered: {
			newConversationWindow.sendMessage(toField.text, subjectField.text, messageArea.text)
		}
	}

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
			text: "Write message here. Ctrl + Enter to send potatoes."
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

	    TableView {
	        id: tableView

	        frameVisible: false
	        sortIndicatorVisible: true

	        anchors.fill: parent

	        Layout.minimumWidth: 400
	        Layout.minimumHeight: 240
	        Layout.preferredWidth: 600
	        Layout.preferredHeight: 400

	        TableViewColumn {
	            id: titleColumn
	            title: "Title"
	            role: "title"
	            movable: false
	            resizable: false
	            width: tableView.viewport.width - authorColumn.width
	        }

	        TableViewColumn {
	            id: authorColumn
	            title: "Author"
	            role: "author"
	            movable: false
	            resizable: false
	            width: tableView.viewport.width / 3
	        }

	        model: ListModel {
	            id: sourceModel
	        	objectName: "listModel"
	        	
	        	function addItem(json) {
	        		append(JSON.parse(json))
	        	}

	            Component.onCompleted: {
			        append({
		                title: "Death of a Salesman",
		                author: "Arthur Miller"
					})
					for (var i=0; i < library.count; i++) {
			        	append({
		                	title: book.title,
		                	author: book.author
		            	})
			    	}
			    }
	        }
	    }
    }
}
