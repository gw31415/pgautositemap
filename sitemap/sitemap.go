package sitemap

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gw31415/pgautositemap/utils"
)

type SitemapManager interface {
	ChannelCreateHandler(s *discordgo.Session, ch *discordgo.ChannelCreate)
	ChannelUpdateHandler(s *discordgo.Session, ch *discordgo.ChannelUpdate)
	ChannelDeleteHandler(s *discordgo.Session, ch *discordgo.ChannelDelete)
	GuildCreateHandler(s *discordgo.Session, g *discordgo.GuildCreate)
	GuildUpdateHandler(s *discordgo.Session, g *discordgo.GuildUpdate)
}

type channelData struct {
	// チャンネルID
	id string
	// チャンネル説明
	topic string
}

type smManager struct {
	mu      sync.Mutex
	targets []string
	timer   *time.Timer

	// サーバーID
	guildID string
	// サイトマップカテゴリID
	sitemapCategoryID string
	// サイトマップのキャッシュ
	smOlds []string
}

func (m *smManager) handler(s *discordgo.Session, target []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 蓄積
	if target == nil {
		m.targets = nil
	} else if m.targets != nil {
		m.targets = append(m.targets, target...)
	}

	// タイマーが存在する場合はリセット
	if m.timer != nil {
		m.timer.Stop()
	}

	// タイマーを1秒後に設定
	m.timer = time.AfterFunc(1*time.Second, func() {
		m.mu.Lock()
		targets := m.targets
		m.targets = []string{}
		m.mu.Unlock()
		m.createSitemaps(s, targets)
	})
}

const (
	// チャンネルの作成
	actionTypeChannelCreate = iota
	// チャンネル位置の変更
	actionTypeChannelMove
	// チャンネルの削除
	actionTypeChannelDelete
	// メッセージを更新する
	actionTypeRefreshMessage
)

type action struct {
	// 作成、更新、削除
	actionType int
	// ID
	id string
	// チャンネル名
	name string
	// コンテンツ
	content string
	// 位置
	position int
}

func (a *action) do(s *discordgo.Session, guildID string, sitemapCategoryID string) {
	switch a.actionType {
	case actionTypeChannelCreate:
		ch, err := s.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
			Name:     a.name,
			Type:     discordgo.ChannelTypeGuildText,
			Position: a.position,
			ParentID: sitemapCategoryID,
		})
		if err != nil {
			slog.Error("Failed to create channel", "error", err)
		}
		if _, err := s.ChannelMessageSend(ch.ID, a.content); err != nil {
			slog.Error("Failed to send message", "error", err)
		}
	case actionTypeChannelDelete:
		_, err := s.ChannelDelete(a.id)
		if err != nil {
			slog.Error("Failed to delete channel", "error", err)
		}
	case actionTypeChannelMove:
		_, err := s.ChannelEditComplex(a.id, &discordgo.ChannelEdit{
			Position: &a.position,
		})
		if err != nil {
			slog.Error("Failed to move channel", "error", err)
		}
	case actionTypeRefreshMessage:

		lastMsgs, err := s.ChannelMessages(a.id, 1, "", "", "")
		if err != nil {
			slog.Error("Failed to get messages", "error", err)
			return
		}
		var lastMsg *discordgo.Message = nil
		lastID := ""
		if len(lastMsgs) == 1 {
			lastMsg = lastMsgs[0]
			lastID = lastMsg.ID
		}

		// メッセージを最新以外全て削除
		for {
			ms, err := s.ChannelMessages(a.id, 100, lastID, "", "")
			// ms, err := s.ChannelMessages(a.id, 100, "", "", "")
			if err != nil {
				slog.Error("Failed to get messages", "error", err)
				break
			}
			msgs := []string{}
			for _, m := range ms {
				msgs = append(msgs, m.ID)
			}
			err = s.ChannelMessagesBulkDelete(a.id, msgs)
			if err != nil {
				slog.Error("Failed to delete messages", "error", err)
				break
			}
			if len(ms) < 100 {
				break
			}
		}

		if lastMsg == nil || lastMsg.Content[:hashLength] != a.content[:hashLength] {
			// 最新のメッセージが存在しないか、内容のハッシュが異なる場合
			// メッセージを送信
			s.ChannelMessageDelete(a.id, lastID)
			_, err = s.ChannelMessageSend(a.id, a.content)
			if err != nil {
				slog.Error("Failed to send message", "error", err)
			}
		}
	}
}

// コースマネージャを生成
func NewSitemapManager(guildID, sitemapCategoryID string) SitemapManager {
	return &smManager{
		guildID:           guildID,
		sitemapCategoryID: sitemapCategoryID,
	}
}

func (m *smManager) createSmName(ch *discordgo.Channel) string {
	lower := strings.ToLower(ch.Name)
	return fmt.Sprintf("sm-%s", lower)
}

// サーバーのロール情報を同期
func (m *smManager) createSitemaps(s *discordgo.Session, targets []string) {
	onlySmChs := false
	for _, target := range targets {
		if target == m.sitemapCategoryID || slices.Contains(m.smOlds, target) {
			onlySmChs = true
			break
		}
	}
	if onlySmChs {
		return
	}

	channels, err := s.GuildChannels(m.guildID)
	if err != nil {
		slog.Error("Failed to get channels", "error", err)
	}

	// サイトマップカテゴリを取得
	var root *discordgo.Channel = nil
	// 既にサイトマップとして使われているチャンネルを取得
	smOlds := []*discordgo.Channel{}
	// サイトマップにするカテゴリのチャンネルIDを取得
	cateChs := []*discordgo.Channel{}
	for _, ch := range channels {
		if ch.ID == m.sitemapCategoryID {
			root = ch
		} else if ch.ParentID == m.sitemapCategoryID {
			smOlds = append(smOlds, ch)
			m.smOlds = append(m.smOlds, ch.ID)
		} else if ch.Type == discordgo.ChannelTypeGuildCategory {
			cateChs = append(cateChs, ch)
		}
	}
	if root == nil {
		slog.Error("Failed to get sitemap category")
		return
	}
	slices.SortFunc(cateChs, func(a, b *discordgo.Channel) int {
		return a.Position - b.Position
	})

	// カテゴリの子チャンネルのマップを作成
	// ID string -> children []*Channel
	tree := make(map[string][]*discordgo.Channel)
	for _, ch := range channels {
		if slices.ContainsFunc(cateChs, func(c *discordgo.Channel) bool {
			return c.ID == ch.ParentID
		}) {
			if children, ok := tree[ch.ParentID]; ok {
				tree[ch.ParentID] = append(children, ch)
			} else {
				tree[ch.ParentID] = []*discordgo.Channel{ch}
			}
		}
	}
	// Positionの昇順に並べ替え
	for _, chs := range tree {
		slices.SortFunc(chs, func(a, b *discordgo.Channel) int {
			return a.Position - b.Position
		})
	}

	// 子チャンネルのないカテゴリを削除
	cateChs = utils.Filter(cateChs, func(c *discordgo.Channel) bool {
		return len(tree[c.ID]) > 0
	})

	// サイトマップのメッセージの作成
	// Name string -> Content string
	smNames := []string{}
	id2RelatedName := make(map[string]string)
	smContents := make(map[string]string)
	for _, cate := range cateChs {

		// チャンネル名の重複チェック
		name := m.createSmName(cate)
		id2RelatedName[cate.ID] = name
		for _, ch := range tree[cate.ID] {
			id2RelatedName[ch.ID] = name
		}
		if slices.Contains(smNames, name) {
			slog.Error("Duplicate sitemap channel name", "name", name)
			return
		}
		smNames = append(smNames, name)

		// サイトマップのコンテンツを作成
		children := tree[cate.ID]
		sm, err := getHash(children)
		if err != nil {
			slog.Error("Failed to get hash", "error", err)
			return
		}
		sm += "\n"
		for _, child := range children {
			link := fmt.Sprintf("<#%s>", child.ID)
			topic := child.Topic
			sm += fmt.Sprintf("- %s : %s\n", link, topic)
		}
		smContents[m.createSmName(cate)] = sm[:len(sm)-1] // INFO: 最後の改行を削除している
	}

	actions := []*action{}
	{
		smOldNameMap := make(map[string]*discordgo.Channel)
		smOldNames := []string{}
		smPositions := make([]int, len(smNames))
		for _, ch := range smOlds {
			name := ch.Name
			if i := slices.Index(smNames, name); i != -1 {
				smOldNames = append(smOldNames, name)
				smOldNameMap[name] = ch
				smPositions[i] = ch.Position
			} else {
				actions = append(actions, &action{
					actionType: actionTypeChannelDelete,
					id:         ch.ID,
				})
			}
		}
		smPositionsDelta := utils.AdjustPositions(smPositions)

		// できておいてほしいチャンネル・今のチャンネルの2つから作成、更新、削除のアクションを作成
		crt, msg, del := utils.AXorB(smNames, smOldNames)
		actions = append(actions, utils.Map(crt, func(name string) *action {
			i := slices.Index(smNames, name)
			position := smPositionsDelta[i]
			smPositionsDelta[i] = 0
			return &action{
				actionType: actionTypeChannelCreate,
				name:       name,
				content:    smContents[name],
				position:   position,
			}
		})...)
		actions = append(actions, utils.Map(msg, func(name string) *action {
			return &action{
				actionType: actionTypeRefreshMessage,
				name:       name, // INFO: この値は動作には使われないがtargetsによるフィルターのために使われる
				id:         smOldNameMap[name].ID,
				content:    smContents[name],
			}
		})...)
		actions = append(actions, utils.Map(del, func(name string) *action {
			return &action{
				actionType: actionTypeChannelDelete,
				id:         smOldNameMap[name].ID,
			}
		})...)
		for i, pos := range smPositionsDelta {
			if pos != 0 {
				name := smNames[i]
				actions = append(actions, &action{
					actionType: actionTypeChannelMove,
					id:         smOldNameMap[name].ID,
					position:   pos,
				})
			}
		}
	}
	if targets != nil {
		relatedNames := make(map[string]bool)
		for _, target := range targets {
			if name, ok := id2RelatedName[target]; ok {
				relatedNames[name] = true
			}
		}
		// 更新のアクションを絞り込む
		actions = utils.Filter(actions, func(a *action) bool {
			if a.actionType == actionTypeChannelCreate || a.actionType == actionTypeRefreshMessage {
				return relatedNames[a.name]
			}
			return true
		})
	}

	for _, a := range actions {
		a.do(s, m.guildID, m.sitemapCategoryID)
	}
}

func (m *smManager) ChannelCreateHandler(s *discordgo.Session, ch *discordgo.ChannelCreate) {
	m.handler(s, []string{ch.Channel.ID})
}

func (m *smManager) ChannelUpdateHandler(s *discordgo.Session, ch *discordgo.ChannelUpdate) {
	m.handler(s, []string{ch.Channel.ID})
}

func (m *smManager) ChannelDeleteHandler(s *discordgo.Session, ch *discordgo.ChannelDelete) {
	m.handler(s, []string{ch.Channel.ID})
}

func (m *smManager) GuildCreateHandler(s *discordgo.Session, g *discordgo.GuildCreate) {
	m.handler(s, nil)
}

func (m *smManager) GuildUpdateHandler(s *discordgo.Session, g *discordgo.GuildUpdate) {
	m.handler(s, nil)
}

const hashLength = 6

func getHash(a any) (string, error) {
	var b bytes.Buffer
	err := gob.NewEncoder(&b).Encode(a)
	if err != nil {
		return "", errors.New("Failed to encode")
	}
	h := fnv.New32a()
	h.Write(b.Bytes())
	return fmt.Sprintf("%x", h.Sum32())[:hashLength], nil
}
