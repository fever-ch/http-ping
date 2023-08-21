// Copyright 2022-2023 - Raphaël P. Barazzutti
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"fmt"
	"github.com/spf13/cobra"
)

type ConsoleLogger interface {
	Printf(format string, a ...any) (int, error)
}

func NewConsoleBasicLogger() ConsoleLogger {
	return &consoleLoggerImpl{}
}

type consoleLoggerImpl struct {
}

func (logger *consoleLoggerImpl) Printf(format string, a ...any) (int, error) {
	return fmt.Printf(format, a...)
}

func NewConsoleCobraLogger(command *cobra.Command) ConsoleLogger {
	return &consoleLoggerCmdImpl{cmd: command}
}

type consoleLoggerCmdImpl struct {
	cmd *cobra.Command
}

func (logger *consoleLoggerCmdImpl) Printf(format string, a ...any) (int, error) {
	logger.cmd.Printf(format, a...)
	return 0, nil
}
