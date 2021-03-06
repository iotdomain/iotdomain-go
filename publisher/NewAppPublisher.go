package publisher

import (
	"github.com/iotdomain/iotdomain-go/lib"
	"github.com/iotdomain/iotdomain-go/messaging"
)

// NewAppPublisher function for all the boilerplate. This:
//  1. Loads messenger config and create messenger instance
//  2. Load PublisherConfig from <appID>.yaml
//  3. Load appconfig from <appID>.yaml (yes same file)
//  4. Create a publisher using the domain from messenger config and publisherID from <appID>.yaml
//  5. Set to persist nodes and load previously saved nodes
//
//  - appID is the application ID, used as publisher ID unless overridden in <appID>.yaml.
//  - configFolder contains the identity, messenger and application configuration
//     Use "" for default location (~/.config/iotdomain).
//  - cacheFolder contains the saved discovered nodes and publishers files.
//     Use "" for default location (~/.cache/iotdomain).
//  - appConfig optional application object to load <appID>.yaml configuration into
//  - cacheDiscovery loads and saves discovered publisher identities and nodes from cache
//
// This returns publisher instance or error if messenger fails to load
func NewAppPublisher(appID string, configFolder string, appConfig interface{},
	cacheFolder string, cacheDiscovery bool) (*Publisher, error) {

	// 1: load messenger config shared with other publishers
	var messengerConfig = messaging.MessengerConfig{}
	err := lib.LoadMessengerConfig(configFolder, &messengerConfig)
	messenger := messaging.NewMessenger(&messengerConfig)

	// 2: load Publisher config fields from appconfig
	pubConfig := &PublisherConfig{
		SaveDiscoveredNodes:      cacheDiscovery,
		SaveDiscoveredPublishers: cacheDiscovery,
		ConfigFolder:             configFolder,
		CacheFolder:              lib.DefaultCacheFolder,
		Loglevel:                 "warning",
		Domain:                   messengerConfig.Domain,
		PublisherID:              appID,
	}
	lib.LoadAppConfig(configFolder, appID, &pubConfig)

	// 3: load application configuration itself
	if appConfig != nil {
		lib.LoadAppConfig(configFolder, appID, appConfig)
	}
	// 4: create the publisher. Reload its identity if available.
	pub := NewPublisher(pubConfig, messenger)

	return pub, err
}
