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

	    TableView {
	        id: tableView
	        objectName: "table"

	        frameVisible: false
	        sortIndicatorVisible: true

	        anchors.fill: parent
	        model: sourceModel

	        Layout.minimumWidth: 400
	        Layout.minimumHeight: 240
	        Layout.preferredWidth: 600
	        Layout.preferredHeight: 400

	        onDoubleClicked: {
	        	var component = Qt.createComponent("new-conversation.qml");
            	component.createObject(historyWindow).show();
	        }

	        TableViewColumn {
	            id: subjectColumn
	            title: "Subject"
	            role: "Subject"
	            movable: false
	            resizable: false
	            width: tableView.viewport.width / 4
	        }

	        TableViewColumn {
	            id: usersColumn
	            title: "Participants"
	            role: "Users"
	            movable: false
	            resizable: false
	            width: tableView.viewport.width / 4
	        }

	        TableViewColumn {
	            id: lastMessageColumn
	            title: "Last Message"
	            role: "LastMessage"
	            movable: false
	            resizable: false
	            width: tableView.viewport.width - usersColumn.width - subjectColumn.width
	        }
 
	    }
    }
}
