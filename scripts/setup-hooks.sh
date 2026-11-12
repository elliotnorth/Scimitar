#!/bin/sh
git -C "$(dirname "$0")/.." config core.hooksPath hooks
