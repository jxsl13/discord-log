package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"syscall"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/handler"
	"github.com/jxsl13/cli-config-boilerplate/cliconfig"
	"github.com/jxsl13/discord-log/config"
	"github.com/spf13/cobra"
	"gopkg.in/Graylog2/go-gelf.v1/gelf"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cmd := NewRootCmd(ctx)
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func NewRootCmd(ctx context.Context) *cobra.Command {
	cctx, cancelCause := context.WithCancelCause(ctx)
	cli := &CLI{
		ctx:         cctx,
		CancelCause: cancelCause,
		cfg:         config.NewConfig(),
	}

	cmd := cobra.Command{
		Use: filepath.Base(os.Args[0]),
	}
	cmd.PreRunE = cli.PreRunE(&cmd)
	cmd.RunE = cli.RunE
	cmd.PostRunE = cli.PostRunE
	return &cmd
}

type CLI struct {
	ctx         context.Context
	CancelCause context.CancelCauseFunc
	cfg         config.Config
	slogger     *slog.Logger
	cleanup     []func()
}

func (cli *CLI) PreRunE(cmd *cobra.Command) func(*cobra.Command, []string) error {
	parser := cliconfig.RegisterFlags(&cli.cfg, false, cmd)
	return func(cmd *cobra.Command, args []string) error {
		log.SetOutput(cmd.OutOrStdout()) // redirect log output to stderr
		err := parser()                  // parse registered commands
		if err != nil {
			return err
		}

		log.Printf("connecting to graylog: %s", cli.cfg.GraylogAddress)
		gelfWriter, err := gelf.NewWriter(cli.cfg.GraylogAddress)
		if err != nil {
			return fmt.Errorf("failed to create gelf writer: %w", err)
		}
		cli.cleanup = append(cli.cleanup, func() {
			err := gelfWriter.Close()
			if err != nil {
				log.Printf("failed to close gelf writer: %v", err)
			}
		})

		handler := slog.NewJSONHandler(io.MultiWriter(cmd.OutOrStdout(), gelfWriter),
			&slog.HandlerOptions{
				ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
					if a.Key == "level" {
						a.Key = "severity"
						return a
					}
					return a
				},
			})
		cli.slogger = slog.New(handler)
		return nil
	}
}

func (cli *CLI) PostRunE(*cobra.Command, []string) error {
	cli.CancelCause(context.Canceled) // cleanup only

	slices.Reverse(cli.cleanup)
	for _, f := range cli.cleanup {
		f()
	}
	return nil
}

func (cli *CLI) RunE(cmd *cobra.Command, args []string) error {

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	s := state.New(cli.cfg.DiscordToken)

	h := handler.New()
	h.AddHandler(func(c *gateway.ReadyEvent) {
		log.Print("connected")
	})
	h.AddHandler(func(c *gateway.MessageDeleteEvent) {
		// Grab from the state
		m, err := s.Message(c.ChannelID, c.ID)
		if err != nil {
			cli.LogDelete(ctx, c.ID, c.ChannelID, c.GuildID)
		} else {
			cli.LogMessage(ctx, m, "delete")
		}
	})

	h.AddHandler(func(c *gateway.MessageCreateEvent) {
		cli.LogMessage(ctx, &c.Message, "create")
	})

	h.AddHandler(func(c *gateway.MessageUpdateEvent) {
		cli.LogMessage(ctx, &c.Message, "update")
	})

	h.AddHandler(func(c *gateway.MessageDeleteBulkEvent) {
		for _, id := range c.IDs {
			m, err := s.Message(c.ChannelID, id)
			if err != nil {
				cli.LogDelete(ctx, id, c.ChannelID, c.GuildID)
			} else {
				cli.LogMessage(ctx, m, "delete")
			}
		}
	})

	s.Handler = h

	log.Printf("connecting to discord...")
	if err := s.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	log.Print("closed")
	return nil
}

func (cli *CLI) LogMessage(ctx context.Context, m *discord.Message, action string) {
	cli.slogger.Log(ctx, slog.LevelInfo,
		action,
		slog.String("action", action),
		slog.Uint64("id", uint64(m.ID)),
		slog.Uint64("channel", uint64(m.ChannelID)),
		slog.Uint64("guild", uint64(m.GuildID)),
		slog.Uint64("type", uint64(m.Type)),
		slog.Int64("flags", int64(m.Flags)),
		slog.Time("timestamp", m.Timestamp.Time()),
		slog.Time("edited", m.EditedTimestamp.Time()),
		slog.String("author", m.Author.Username),
		slog.String("content", m.Content),
	)
}

func (cli *CLI) LogDelete(ctx context.Context, messageID discord.MessageID, channelID discord.ChannelID, guildID discord.GuildID) {
	cli.slogger.Log(ctx, slog.LevelInfo,
		"delete",
		slog.String("action", "delete"),
		slog.Uint64("id", uint64(messageID)),
		slog.Uint64("channel", uint64(channelID)),
		slog.Uint64("guild", uint64(guildID)),
	)
}
