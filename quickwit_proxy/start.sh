#!/bin/sh

quickwit run --config /quickwit/config/config.yaml &
/quickwit/config/proxy
