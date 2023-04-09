package store

/*
id bigint, msg_id(unique) bigint, parent_id bigint, session_id bigint
*/

/*
CREATE TABLE IF NOT EXISTS MSG_PARENT (
    id integer,
    msg_id varchar(255),
    parent_id varchar(255),
    session_id varchar(255),
    PRIMARY KEY (id),
    UNIQUE (msg_id)

CREATE INDEX idx_msg_id ON MSG_PARENT (msg_id);
CREATE INDEX idx_session_id ON MSG_PARENT (session_id);
*/

import (
	"context"

	"github.com/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type MsgRoot struct {
	ID        int64 `gorm:"primaryKey"`
	MsgID     string
	ParentID  string
	SessionID string
}

var dbMsg *gorm.DB

func init() {
	var err error

	dbMsg, err = gorm.Open(sqlite.Open("db/msg.db"), &gorm.Config{})

	if err != nil {
		panic("failed to init sqlite, err: " + err.Error())
	}
}

// This function retrieves the session ID of a message based on the message ID.
// If the message is a reply to another message, the function retrieves the session ID of the original message.
// If the message is not a reply to another message, the function returns the message ID as the session ID.
func GetMsgSessionID(ctx context.Context, msgID string) (parentID string, err error) {
	rootMsg := MsgRoot{}
	r := dbMsg.Debug().WithContext(ctx).Table("MSG_PARENT").Where("msg_id = ?", msgID).First(&rootMsg)
	if r.Error == nil {
		return rootMsg.SessionID, nil
	} else {
		return "", errors.Wrapf(r.Error, "failed to get msg parent id, msg_id: %v", msgID)
	}
}

// This function creates a record in the MsgRoot table for a message.
// If the message is a reply to another message, the function creates a record in the MsgRoot table for the original message.
// If the message is not a reply to another message, the function creates a record in the MsgRoot table for the message.
func ReceiveMessage(ctx context.Context, msgID string, replyMsgID string) (sessionID string, err error) {
	rootID := msgID
	if replyMsgID != "" {
		rootID, err = GetMsgSessionID(ctx, replyMsgID)
		if err != nil && err != gorm.ErrRecordNotFound {
			return "", errors.Wrapf(err, "failed to get msg session id, msg_id: %v, parent_id: %v", msgID, replyMsgID)
		}
	}
	rootMsg := MsgRoot{
		MsgID:     msgID,
		ParentID:  replyMsgID,
		SessionID: rootID,
	}
	r := dbMsg.Debug().WithContext(ctx).Table("MSG_PARENT").Create(&rootMsg)
	if r.Error != nil {
		return "", errors.Wrapf(r.Error, "failed to create msg root, msg_id: %v, parent_id: %v, rootID:%v ", msgID, replyMsgID, rootID)
	}
	return rootMsg.SessionID, nil
}
