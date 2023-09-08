package tg_service

import (
	"fmt"
	"myapp/internal/models"
	"myapp/internal/service/tg_service"
	u "myapp/internal/utils"
	"time"

	"go.uber.org/zap"
)

func (srv *tg_service.TgService) Donor_HandleEditEditedChannelPost(m models.Update) error {
	chatId := m.EditedChannelPost.Chat.Id
	// msgText := m.Message.Text
	// userFirstName := m.Message.From.FirstName
	// userUserName := m.Message.From.UserName
	srv.l.Info("tgClient: Donor_EditEditedChannelPost", zap.Any("models.Update", m))

	err := srv.Donor_editEditedChannelPost(m)
	if err != nil {
		if err != nil {
			srv.ShowMessClient(chatId, u.ERR_MSG_2+err.Error())
		}
		return err
	}
	return nil
}

func (srv *tg_service.TgService) Donor_editEditedChannelPost(m models.Update) error {
	// chatId := m.Message.Chat.ID
	// msgText := m.Message.Text
	// userFirstName := m.Message.From.FirstName
	// userUserName := m.Message.From.UserName
	// srv.l.Info("tg_service::AddEditedChannelPost::")

	message_id := m.EditedChannelPost.MessageId

	// Проверка что пост есть уже в базе нужна для того что бы телега не отрпавляла
	// кучу запросов повторно , тк ответ долгий из за рассылки

	// если Media_Group
	if m.EditedChannelPost.MediaGroupId != nil {
		var postType string
		if len(m.EditedChannelPost.Photo) > 0 {
			postType = "photo"
		} else if m.EditedChannelPost.Video.FileId != "" {
			postType = "video"
		} else {
			return fmt.Errorf("Media_Group без photo и video")
		}
		filePath, err := srv.downloadPostMedia(m, postType)
		if err != nil {
			return err
		}
		newmedia := tg_service.Media{
			Media_group_id:            *m.EditedChannelPost.MediaGroupId,
			Type_media:                postType,
			fileNameInServer:          filePath,
			Donor_message_id:          message_id,
			Reply_to_donor_message_id: 0,
			Caption:                   "",
			Caption_entities:          m.EditedChannelPost.CaptionEntities,
			//File_id: // нужно для подтверждения в доноре, позже в вампирах заменяем
			//Reply_to_message_id:  // нужно для подтверждения в доноре, позже в вампирах заменяем
		}
		if postType == "photo" {
			newmedia.File_id = m.EditedChannelPost.Photo[len(m.EditedChannelPost.Photo)-1].FileId
		} else if postType == "video" {
			newmedia.File_id = m.EditedChannelPost.Video.FileId
		}
		if m.EditedChannelPost.ReplyToMessage != nil {
			newmedia.Reply_to_message_id = m.EditedChannelPost.ReplyToMessage.MessageId
			newmedia.Reply_to_donor_message_id = m.EditedChannelPost.ReplyToMessage.MessageId
		}
		if m.EditedChannelPost.Caption != nil {
			newmedia.Caption = *m.EditedChannelPost.Caption
		}

		srv.MediaCh <- newmedia
		return nil
	}

	// если не Media_Group
	allVampBots, err := srv.As.GetAllVampBots()
	if err != nil {
		return err
	}
	for i, vampBot := range allVampBots {
		if vampBot.ChId == 0 {
			continue
		}
		err := srv.editChPostAsVamp(vampBot, m)
		if err != nil {
			srv.l.Error("Donor_EditChannelPost: editChPostAsVamp", zap.Error(err))
		}
		srv.l.Info("Donor_EditChannelPost", zap.Any("bot index in arr", i), zap.Any("bot ch link", vampBot.ChLink))
		time.Sleep(time.Second * 2)
	}

	return nil
}
