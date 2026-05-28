#!/bin/bash
# Add WEBHOOK_SECRET to .env if not present
grep -q WEBHOOK_SECRET /Users/home/git/kms-server/.env || echo "WEBHOOK_SECRET=kms-wh-se...cret" >> /Users/home/git/kms-server/.env
