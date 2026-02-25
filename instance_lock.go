package main

import "errors"

var ErrAlreadyRunning = errors.New("another instance is already running")
