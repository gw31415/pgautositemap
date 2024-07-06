package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

var (
	// デバッグモード
	DEBUG_MODE = len(os.Getenv("DEBUG_MODE")) > 0

	// Discordのトークン
	DISCORD_TOKEN = os.Getenv("DISCORD_TOKEN")

	// DiscordサーバーID
	GUILD_ID = os.Getenv("GUILD_ID")
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
		slog.Error("Error creating Discord session:", err)
		return
	}
	discord.Identify.Intents = discordgo.IntentsGuildMembers | discordgo.IntentsGuilds

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

	// Discordセッションの開始
	slog.Info("Opening discord connection")
	err = discord.Open()
	if err != nil {
		slog.Error("Error opening discord connection:", err)
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
