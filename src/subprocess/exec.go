package subprocess

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/aurora-is-near/remaphore/src/protocol"
	"github.com/btcsuite/btcutil/base58"
)

func Exec(ctx context.Context, config *protocol.Config, args []string, msg *protocol.Message) (out string, err error) {
	var destMatch string
	pubkey := base58.Encode(msg.SenderPublicKey)
	args = append(args, msg.Verb, msg.Payload)
	destMatches := protocol.MatchWildcards(config.Destination, msg.Destination)
	if destMatches {
		destMatch = config.Destination
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = []string{
		fmt.Sprintf("%s=%s", "REMAPHORE_SENDER", pubkey),
		fmt.Sprintf("%s=%s", "REMAPHORE_VERB", msg.Verb),
		fmt.Sprintf("%s=%d", "REMAPHORE_TIME", msg.SendTimeNano/int64(time.Second)),
		fmt.Sprintf("%s=%x", "REMAPHORE_UUID", msg.UUID),
		fmt.Sprintf("%s=%s", "REMAPHORE_MSG", msg.Payload),
	}
	if destMatches {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", "REMAPHORE_DESTMATCH", destMatch))
	}
	op1, err := cmd.CombinedOutput()
	exitCode := cmd.ProcessState.ExitCode()
	log.Printf("Exec (%d): '%s'", exitCode, strings.Join(args, " "))
	return fmt.Sprintf("%d,%s", exitCode, string(op1)), err
}
