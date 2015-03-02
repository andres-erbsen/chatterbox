import QtQuick 2.2
import QtQuick.Controls 1.1
import QtQuick.Layouts 1.1

ApplicationWindow {
	id: historyWindow

    visible: true
    title: "History"
    property int margin: 5
    width: mainLayout.implicitWidth + 2 * margin
    height: mainLayout.implicitHeight + 2 * margin
    minimumWidth: mainLayout.Layout.minimumWidth + 40 * margin
    minimumHeight: mainLayout.Layout.minimumHeight + 12 * margin

	ListModel {
	    id: sourceModel
		objectName: "listModel"

		function addItem(json) {
			var parsed = JSON.parse(json);
			// TODO represents participants using some QML-(color?)-delimited thing, comma-separated encoding is not reversible
			append({Subject: parsed.Subject, Participants:parsed.Participants.toString()});
		}
	}


    ColumnLayout {
        id: mainLayout
        anchors.fill: parent
        anchors.margins: margin

	    TableView {
	        id: tableView
	        objectName: "table"

	        focus:true
	        frameVisible: true
	        sortIndicatorVisible: false

	        model: sourceModel
			Layout.fillHeight: true
			Layout.fillWidth: true

	        TableViewColumn {
	            id: usersColumn
	            title: "Participants"
	            role: "Participants"
	            movable: false
	        }

	        TableViewColumn {
	            id: subjectColumn
	            title: "Subject"
	            role: "Subject"
	            movable: false
	        }
	    }

		Button {
			id: newConversationButton
	        objectName: "newConversationButton"
			action: newConversation
		}
    }

	Action {
		id: newConversation
		objectName: "newConversation"
		text: "&New Conversation"
		shortcut: "Ctrl+N"
	}
}
