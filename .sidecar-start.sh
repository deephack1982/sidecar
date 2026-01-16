#!/bin/bash
PROMPT=$(cat "/Users/marcusvorwaller/code/review-session/.sidecar-prompt")
rm -f "/Users/marcusvorwaller/code/review-session/.sidecar-prompt"
claude --dangerously-skip-permissions "$PROMPT"
rm -f "/Users/marcusvorwaller/code/review-session/.sidecar-start.sh"
