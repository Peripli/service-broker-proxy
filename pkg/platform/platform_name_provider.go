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

package platform

// PlatformNameProvider provides the resource name in the platform,
// according to how the platform enforces it (e.g k8s not allowing upper case letters)
//go:generate counterfeiter . BrokerPlatformNameProvider
type BrokerPlatformNameProvider interface {
	// GetBrokerPlatformName obtains the broker name in the platform
	GetBrokerPlatformName(name string) string
}
