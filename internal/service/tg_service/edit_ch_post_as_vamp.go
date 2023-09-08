package tg_service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"myapp/internal/entity"
	"myapp/internal/models"
	"myapp/internal/repository"
	"myapp/internal/service/tg_service"
	"myapp/pkg/mycopy"
	"net/http"
	"strconv"
	"strings"
)

func (srv *tg_service.TgService) editChPostAsVamp(vampBot entity.Bot, m models.Update) error {
	donor_ch_mes_id := m.EditedChannelPost.MessageId
	if m.EditedChannelPost.Text == "DeletePost" {
		currPost, err := srv.As.GetPostByDonorIdAndChId(donor_ch_mes_id, vampBot.ChId)
		if err != nil {
			return fmt.Errorf("sendChPostAsVamp (1): %v", err)
		}
		messageForDelete := currPost.PostId
		DelJson, err := json.Marshal(map[string]any{
			"chat_id":    strconv.Itoa(vampBot.ChId),
			"message_id": strconv.Itoa(messageForDelete),
		})
		if err != nil {
			return err
		}
		rrres, err := http.Post(
			fmt.Sprintf(srv.TgEndp, vampBot.Token, "deleteMessage"),
			"application/json",
			bytes.NewBuffer(DelJson),
		)
		if err != nil {
			return err
		}
		defer rrres.Body.Close()
	}
	if m.EditedChannelPost.VideoNote != nil {
		//////////////// если кружочек видео
		return nil
	} else if len(m.EditedChannelPost.Photo) > 0 {
		//////////////// если фото
		return nil
	} else if m.EditedChannelPost.Video != nil {
		//////////////// если видео
		return nil
	} else {
		//////////////// если просто текст
		futureMesJson := map[string]any{
			"chat_id": strconv.Itoa(vampBot.ChId),
		}
		currPost, err := srv.As.GetPostByDonorIdAndChId(donor_ch_mes_id, vampBot.ChId)
		if err != nil {
			return fmt.Errorf("sendChPostAsVamp (1): %v", err)
		}
		futureMesJson["message_id"] = currPost.PostId

		var messText string // строка в которую скопируем значение текста поста, тк структуры копируются по ебаной ссылке, и если срезаем часть текста то потом везде так будет
		if len(m.EditedChannelPost.Entities) > 0 {
			entities := make([]models.MessageEntity, len(m.EditedChannelPost.Entities))
			mycopy.DeepCopy(m.EditedChannelPost.Entities, &entities)
			cutEntities := false
			for i, v := range entities {
				if strings.HasPrefix(v.Url, "http://fake-link") || strings.HasPrefix(v.Url, "fake-link") || strings.HasPrefix(v.Url, "https://fake-link") {
					groupLink, err := srv.As.GetGroupLinkById(vampBot.GroupLinkId)
					if err != nil && !errors.Is(err, repository.ErrNotFound) {
						return err
					}
					srv.l.Info("sendChPostAsVamp -> если просто текст -> entities -> GetGroupLinkById", zap.Any("vampBot", vampBot), zap.Any("groupLink", groupLink))
					if groupLink.Link == "" {
						continue
					}
					if strings.HasPrefix(groupLink.Link, "http://cut-link") || strings.HasPrefix(groupLink.Link, "cut-link") || strings.HasPrefix(groupLink.Link, "https://cut-link") {
						mycopy.DeepCopy(m.EditedChannelPost.Text, &messText) // какого хуя в Го структуры копируются по ссылке  ??
						messText = strings.Replace(messText, "Переходим по ссылке - ССЫЛКА", "", -1)
						messText = strings.Replace(messText, "👉 РЕГИСТРАЦИЯ ТУТ 👈", "", -1)
						messText = strings.Replace(messText, "🔖 Написать мне 🔖", "", -1)
						cutEntities = true
						break
					}
					entities[i].Url = groupLink.Link
					continue
				}
				urlArr := strings.Split(v.Url, "/")
				for ii, vv := range urlArr {
					if vv == "t.me" && urlArr[ii+1] == "c" {
						refToDonorChPostId, err := strconv.Atoi(urlArr[ii+3])
						if err != nil {
							return err
						}
						currPost, err := srv.As.GetPostByDonorIdAndChId(refToDonorChPostId, vampBot.ChId)
						if err != nil {
							return fmt.Errorf("sendChPostAsVamp (2): %v", err)
						}
						if vampBot.ChId < 0 {
							urlArr[ii+2] = strconv.Itoa(-vampBot.ChId)
						} else {
							urlArr[ii+2] = strconv.Itoa(vampBot.ChId)
						}
						if urlArr[ii+2][0] == '1' && urlArr[ii+2][1] == '0' && urlArr[ii+2][2] == '0' {
							urlArr[ii+2] = urlArr[ii+2][3:]
						}
						urlArr[ii+3] = strconv.Itoa(currPost.PostId)
						entities[i].Url = strings.Join(urlArr, "/")
					}
				}
			}
			if !cutEntities {
				futureMesJson["entities"] = entities
			}
		}

		text_message := m.EditedChannelPost.Text
		if messText != "" {
			futureMesJson["text"] = messText
		} else {
			futureMesJson["text"] = text_message
		}
		json_data, err := json.Marshal(futureMesJson)
		if err != nil {
			return err
		}
		srv.l.Info("sendChPostAsVamp -> если просто текст -> http.Post", zap.Any("futureMesJson", futureMesJson), zap.Any("string(json_data)", string(json_data)))
		editVampPostResp, err := http.Post(
			fmt.Sprintf(srv.TgEndp, vampBot.Token, "editMessageText"),
			"application/json",
			bytes.NewBuffer(json_data),
		)
		if err != nil {
			return err
		}
		defer editVampPostResp.Body.Close()
		var cAny struct {
			Ok     bool `json:"ok"`
			Result struct {
				MessageId int `json:"message_id"`
			} `json:"result,omitempty"`
		}
		if err := json.NewDecoder(editVampPostResp.Body).Decode(&cAny); err != nil {
			return err
		}
		if cAny.Result.MessageId != 0 {
			if err != nil {
				return err
			}
		}
	}
	return nil
}
