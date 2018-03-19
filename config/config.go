package config

import (
	"io/ioutil"
	"os"
	"path"

	"google.golang.org/appengine/log"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v2"

	"google.golang.org/appengine"
	"google.golang.org/appengine/file"
)

//Config is the in-memory store for the loaded configuration.
var Config map[string]string

//Load uses the suppplies context to load configuration data - a map - based on
//the environement. If the environment is the appengine development server,
//it loads the configuration from the local config.yaml file. If the
//environment is the production appengine server, it loads config.yaml from
//the default Google Cloud Storage bucket.
//Once loaded, the config is stored in memory for quicker access.
func Load(c context.Context) (map[string]string, error) {
	if len(Config) != 0 {
		return Config, nil
	}
	var (
		configFile []byte
		err        error
	)
	if appengine.IsDevAppServer() {
		configFile, err = ioutil.ReadFile(path.Join(os.Getenv("PWD"), "config.yaml"))
	} else {
		storageClient, err := storage.NewClient(c)
		if err != nil {
			return nil, err
		}
		bucket, _ := file.DefaultBucketName(c)
		rc, err := storageClient.Bucket(bucket).Object("config.yaml").NewReader(c)
		if err != nil {
			log.Errorf(c, "error reading config: %v", err.Error())
		}

		configFile, err = ioutil.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
	}
	yaml.Unmarshal(configFile, &Config)
	return Config, err
}
