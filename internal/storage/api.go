// Database Gateway provides access to servers with ACL for safe and restricted database interactions.
// Copyright (C) 2024  Kirill Zhuravlev
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package storage

// type InsertChatReq struct {
//	ID       int
//	Username string
//	FullChat tg.MessagesChatFull
//	Peer     tg.InputPeerChannel
//	MaxMsgID int
//	MinMsgID int
//	IsJoined bool
//}
//
//func (s *Service) InsertChat(conn qrm.DB, req InsertChatReq) error {
//	payloadChat := new(bin.Buffer)
//	if err := req.FullChat.Encode(payloadChat); err != nil {
//		return fmt.Errorf("encode full chat: %w", err)
//	}
//
//	payloadPeer := new(bin.Buffer)
//	if err := req.Peer.Encode(payloadPeer); err != nil {
//		return fmt.Errorf("encode full chat: %w", err)
//	}
//
//	{
//		obj := model.Chats{
//			ID:            int32(req.ID),
//			Username:      req.Username,
//			Peer:          payloadPeer.Buf,
//			Payload:       payloadChat.Buf,
//			LastFetchedAt: time.Now(),
//			MaxMsgID:      int32(req.MaxMsgID),
//			MinMsgID:      int32(req.MinMsgID),
//			IsJoined:      req.IsJoined,
//		}
//		res, err := tbl.Chats.
//			INSERT(tbl.Chats.AllColumns).
//			MODEL(obj).
//			Exec(conn)
//		if err := handleError("insert chat", err, res); err != nil {
//			return err
//		}
//	}
//
//	return nil
//}
//
//func (s *Service) GetChannelByID(conn qrm.DB, id int) (*Chat, error) {
//	var obj model.Chats
//	err := tbl.Chats.
//		SELECT(tbl.Chats.AllColumns).
//		WHERE(tbl.Chats.ID.EQ(postgres.Int(int64(id)))).
//		LIMIT(1).
//		Query(conn, &obj)
//	if err := handleError("get chat by id", err, nil); err != nil {
//		return nil, err
//	}
//
//	return just.Pointer(adaptChat(obj)), nil
//}
//
//func (s *Service) GetChannelByIDs(conn qrm.DB, ids []int) ([]Chat, error) {
//	if len(ids) == 0 {
//		return []Chat{}, nil
//	}
//
//	pgIDs := just.SliceMap(ids, func(id int) postgres.Expression {
//		return postgres.Int(int64(id))
//	})
//
//	var obj []model.Chats
//	err := tbl.Chats.
//		SELECT(tbl.Chats.AllColumns).
//		WHERE(tbl.Chats.ID.IN(pgIDs...)).
//		Query(conn, &obj)
//	if err := handleError("get chat by ids", err, nil); err != nil {
//		return nil, err
//	}
//
//	return adaptChats(obj), nil
//}
//
//func (s *Service) GetChannelByUsername(conn qrm.DB, username string) (*Chat, error) {
//	var obj model.Chats
//	err := tbl.Chats.
//		SELECT(tbl.Chats.AllColumns).
//		WHERE(tbl.Chats.Username.EQ(postgres.String(username))).
//		LIMIT(1).
//		Query(conn, &obj)
//	if err := handleError("get chat by username", err, nil); err != nil {
//		return nil, err
//	}
//
//	return just.Pointer(adaptChat(obj)), nil
//}
//
//func (s *Service) FilterChannelByUsernames(conn qrm.DB, usernames []string) ([]Chat, error) {
//	usernames = just.SliceUniq(usernames)
//	if len(usernames) == 0 {
//		return []Chat{}, nil
//	}
//
//	pgUsernames := just.SliceMap(usernames, func(username string) postgres.Expression {
//		return postgres.String(username)
//	})
//
//	var objects []model.Chats
//	err := tbl.Chats.
//		SELECT(tbl.Chats.AllColumns).
//		WHERE(tbl.Chats.Username.IN(pgUsernames...)).
//		Query(conn, &objects)
//	if err := handleError("get chats by usernames", err, nil); err != nil {
//		return nil, err
//	}
//
//	return adaptChats(objects), nil
//}
//
//type SetChatMinIDReq struct {
//	ChatID   int
//	MinMsgID int
//}
//
//func (s *Service) SetChatMinID(conn qrm.DB, req SetChatMinIDReq) error {
//	_, err := tbl.Chats.
//		UPDATE(tbl.Chats.MinMsgID).
//		SET(tbl.Chats.MinMsgID.SET(postgres.Int(int64(req.MinMsgID)))).
//		WHERE(postgres.AND(
//			tbl.Chats.ID.EQ(postgres.Int(int64(req.ChatID))),
//			postgres.OR(
//				tbl.Chats.MinMsgID.GT(postgres.Int(int64(req.MinMsgID))),
//				tbl.Chats.MinMsgID.EQ(postgres.Int(-1)),
//			),
//		)).
//		Exec(conn)
//	if err != nil {
//		return fmt.Errorf("set chat min id: %w", err)
//	}
//
//	return nil
//}
//
//func (s *Service) TouchChatLastFetchedAt(conn qrm.DB, chatID int) error {
//	res, err := tbl.Chats.
//		UPDATE(tbl.Chats.LastFetchedAt).
//		SET(tbl.Chats.LastFetchedAt.SET(postgres.TimestampzT(time.Now()))).
//		WHERE(tbl.Chats.ID.EQ(postgres.Int(int64(chatID)))).
//		Exec(conn)
//	if err := handleError("set chat LastFetchedAt", err, res); err != nil {
//		return err
//	}
//
//	return nil
//}
//
//func (s *Service) SetChatIsJoined(conn qrm.DB, chatID int, isJoined bool) error {
//	res, err := tbl.Chats.
//		UPDATE(tbl.Chats.IsJoined).
//		SET(tbl.Chats.IsJoined.SET(postgres.Bool(isJoined))).
//		WHERE(tbl.Chats.ID.EQ(postgres.Int(int64(chatID)))).
//		Exec(conn)
//	if err := handleError("set chat isJoined", err, res); err != nil {
//		return err
//	}
//
//	return nil
//}
//
//func (s *Service) GetChannelsRandom(conn qrm.DB, limit int) ([]Chat, error) {
//	var objects []model.Chats
//	err := tbl.Chats.
//		SELECT(tbl.Chats.AllColumns).
//		LIMIT(int64(limit)).
//		Query(conn, &objects)
//	if err := handleError("get rand channels from db", err, nil); err != nil {
//		return nil, err
//	}
//
//	return adaptChats(objects), nil
//}
//
//func (s *Service) GetChannelsAll(conn qrm.DB) ([]Chat, error) {
//	var objects []model.Chats
//	err := tbl.Chats.
//		SELECT(tbl.Chats.AllColumns).
//		ORDER_BY(tbl.Chats.ID.ASC()).
//		Query(conn, &objects)
//	if err := handleError("get all channels from db", err, nil); err != nil {
//		return nil, err
//	}
//
//	return adaptChats(objects), nil
//}
