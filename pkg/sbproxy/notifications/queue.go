/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package notifications

import "github.com/Peripli/service-manager/pkg/types"

type Queue interface {
	Push(*types.Notification)
	Listen() chan *types.Notification
	Clean()
}

type ChannelQueue chan *types.Notification

func (c ChannelQueue) Push(n *types.Notification) {
	c <- n
}

func (c ChannelQueue) Listen() chan *types.Notification {
	return c
}

func (c ChannelQueue) Clean() {
	for {
		select {
		case <-c:
		default:
			return
		}
	}
}
