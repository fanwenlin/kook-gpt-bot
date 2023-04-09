package store

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
)

/*
id bigint, msg_id(unique) bigint, parent_id bigint, session_id bigint
*/

/*
CREATE TABLE IF NOT EXISTS MSG_PARENT (
	id bigint,
	msg_id varchar(255),
	parent_id varchar(255),
	session_id varchar(255),
	PRIMARY KEY (id),
	UNIQUE (msg_id)
);
CREATE INDEX idx_msg_id ON MSG_PARENT (msg_id);
CREATE INDEX idx_session_id ON MSG_PARENT (session_id);
*/

type SessionContext struct {
	SessionID string `json:"session_id" gorm:"primaryKey"`
	MsgList   string `json:"msg_list" gorm:"type:text;not null;default:''"`
}

type Msg struct {
	Content string
	Role    int
}

const (
	RoleSystem = 0
	RoleUser   = 1
	RoleAI     = 2
)

/*
CREATE TABLE IF NOT EXISTS MSG_CONTEXT (
	id bigint,
	session_id varchat(255),
	msg_list text,
	PRIMARY KEY (id),
	UNIQUE (session_id)
);
CREATE INDEX idx_session_id ON MSG_CONTEXT (session_id);
*/

func GetMessageContext(ctx context.Context, sessionID string) (msgList []*Msg, err error) {
	sessionContext := SessionContext{}
	r := dbMsg.Debug().WithContext(ctx).Table("MSG_CONTEXT").Where("session_id = ?", sessionID).First(&sessionContext)
	if r.Error != nil {
		return nil, errors.Wrapf(r.Error, "failed to get msg context, session_id: %v", sessionID)
	}
	msgList = make([]*Msg, 0)
	err = json.Unmarshal([]byte(sessionContext.MsgList), &msgList)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal msg context, session_id: %v", sessionID)
	}

	return msgList, nil
}

func UpdateMessageContext(ctx context.Context, sessionID string, msgList []*Msg) (err error) {
	msgListJson, err := json.Marshal(msgList)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal msg list, session_id: %v", sessionID)
	}
	var session SessionContext
	if err := dbMsg.Debug().Table("MSG_CONTEXT").Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		session = SessionContext{
			SessionID: sessionID,
			MsgList:   string(msgListJson),
		}
	} else {
		session.MsgList = string(msgListJson)
	}
	err = dbMsg.Debug().Table("MSG_CONTEXT").Save(&session).Error
	return err
}
