package main

import (
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/gw31415/pgautositemap/sitemap"
	"github.com/robfig/cron/v3"
)

var (
	// デバッグモード
	DEBUG_MODE = len(os.Getenv("DEBUG_MODE")) > 0

	// DiscordサーバーID
	GUILD_ID = os.Getenv("GUILD_ID")

	// Discordのトークン
	DISCORD_TOKEN = os.Getenv("DISCORD_TOKEN")

	// サイトマップカテゴリID
	SITEMAP_CATEGORY_ID = os.Getenv("SITEMAP_CATEGORY_ID")

	// 差分だけでなく、定期的にサイトマップを強制更新するためのcron設定
	SITEMAP_REFRESHING_CRON = os.Getenv("SITEMAP_REFRESHING_CRON")

	// 無視するカテゴリチャンネルのID, カンマ区切り
	SITEMAP_IGNORE_CATEGORY_CHANNEL = strings.Split(os.Getenv("SITEMAP_IGNORE_CATEGORY_CHANNEL"), ",")
)

func main() {
	if DEBUG_MODE {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		slog.Debug("Debug mode")
	}

	// 環境変数のチェック
	if DISCORD_TOKEN == "" {
		slog.Error("Please set environment variables")
		return
	}

	// Discordセッションの初期化
	discord, err := discordgo.New("Bot " + DISCORD_TOKEN)
	if err != nil {
		slog.Error("Error creating Discord session:", "err", err)
		return
	}
	discord.Identify.Intents = discordgo.IntentsGuilds

	// cronの初期化
	cr := cron.New()

	// 対応外のサーバーから退出する設定
	discord.AddHandler(func(s *discordgo.Session, m *discordgo.GuildCreate) {
		if m.Guild.ID != GUILD_ID {
			slog.Warn("Leaving guild", "GUILD_ID", m.Guild.ID)
			s.GuildLeave(m.Guild.ID)
		}
	})
	discord.AddHandler(func(s *discordgo.Session, m *discordgo.Ready) {
		for _, guild := range m.Guilds {
			if guild.ID != GUILD_ID {
				slog.Warn("Leaving guild", "GUILD_ID", guild.ID)
				s.GuildLeave(guild.ID)
			}
		}
	})

	// サイトマップマネージャーの初期化
	sm := sitemap.NewSitemapManager(GUILD_ID, SITEMAP_CATEGORY_ID, SITEMAP_IGNORE_CATEGORY_CHANNEL)
	discord.AddHandler(sm.GuildCreateHandler)
	discord.AddHandler(sm.GuildUpdateHandler)
	discord.AddHandler(sm.ChannelCreateHandler)
	discord.AddHandler(sm.ChannelUpdateHandler)
	discord.AddHandler(sm.ChannelDeleteHandler)
	_, err = cr.AddFunc(SITEMAP_REFRESHING_CRON, func() {
		slog.Info("Refreshing sitemaps")
		sm.ManuallyUpdate(discord)
	})
	if err != nil {
		slog.Error("Error adding cron job:", "err", err)
		return
	}

	// Discordセッションの開始
	slog.Info("Opening discord connection")
	err = discord.Open()
	if err != nil {
		slog.Error("Error opening discord connection:", "err", err)
		return
	}
	defer discord.Close()

	// cronの開始
	slog.Info("Starting cron")
	go cr.Run()
	defer cr.Stop()

	// 終了シグナルの待機
	slog.Info("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	slog.Info("Shutting down...")
}
