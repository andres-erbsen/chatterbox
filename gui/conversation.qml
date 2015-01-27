import QtQuick 2.2
import QtQuick.Controls 1.1
import QtQuick.Layouts 1.1

ApplicationWindow {
    visible: true
    title: "New Message"
    property int margin: 11
    width: mainLayout.implicitWidth + 2 * margin
    height: mainLayout.implicitHeight + 2 * margin
    minimumWidth: mainLayout.Layout.minimumWidth + 40 * margin
    minimumHeight: mainLayout.Layout.minimumHeight + 12 * margin

    ColumnLayout {
        id: mainLayout
        anchors.fill: parent
        anchors.margins: margin
        GroupBox {
            id: toBox
            title: "To:"
            Layout.fillWidth: true

            RowLayout {
                id: toLayout
                anchors.fill: parent
                TextField {
                    placeholderText: "Comma delimited denames"
                    Layout.fillWidth: true
                }
            }
        }

        GroupBox {
            id: subjectBox
            title: "Subject:"
            Layout.fillWidth: true

            RowLayout {
                id: subjectLayout
                anchors.fill: parent
                TextField {
                    Layout.fillWidth: true
                }
            }
        }

        GroupBox {
            id: messageBox
            title: "Message Contents:"
            Layout.fillWidth: true
            Layout.fillHeight: true

            RowLayout {
                id: textLayout
                anchors.fill: parent
                TextArea {
                    id: messageContents 
                    Layout.minimumHeight: 30
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                }
            }
        }

        GroupBox {
            id: sendMessage
            Layout.alignment: Qt.AlignRight
            flat: true
            Row {
                Button {
                    text: "Send Message"
                }
            }
        }
    }
}
