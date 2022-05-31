package linuxkeyring

import (
	"encoding/json"
	"os"

	"github.com/99designs/keyring"
	"github.com/sirupsen/logrus"
	"github.com/versent/saml2aws/v2/helper/credentials"
)

var logger = logrus.WithField("helper", "linuxkeyring")

type KeyringHelper struct {
	keyring keyring.Keyring
}

func NewKeyringHelper() (*KeyringHelper, error) {
	kr, err := keyring.Open(keyring.Config{
		AllowedBackends: func() []keyring.BackendType {
			switch os.Getenv("SAML2AWS_KEYRING_BACKEND") {
			case "kwallet":
				return []keyring.BackendType{keyring.KWalletBackend}
			case "secret_service":
				return []keyring.BackendType{keyring.SecretServiceBackend}
			case "pass":
				return []keyring.BackendType{keyring.PassBackend}
			default:
				return []keyring.BackendType{
					keyring.KWalletBackend,
					keyring.SecretServiceBackend,
					keyring.PassBackend,
				}
			}
		}(),
		LibSecretCollectionName: "login",
		PassPrefix:              "saml2aws",
		PassDir: func() string {
			passDir := os.Getenv("SAML2AWS_PASSWORD_STORE_DIR")
			if passDir == "" {
				passDir = os.Getenv("HOME") + "/.password_store"
			}
			return passDir
		}(),
	})

	if err != nil {
		return nil, err
	}

	return &KeyringHelper{
		keyring: kr,
	}, nil
}

func (kr *KeyringHelper) Add(creds *credentials.Credentials) error {
	encoded, err := json.Marshal(creds)
	if err != nil {
		return err
	}

	return kr.keyring.Set(keyring.Item{
		Key:                         creds.ServerURL,
		Label:                       credentials.CredsLabel,
		Data:                        encoded,
		KeychainNotTrustApplication: false,
	})
}

func (kr *KeyringHelper) Delete(serverURL string) error {
	return kr.keyring.Remove(serverURL)
}

func (kr *KeyringHelper) Get(serverURL string) (string, string, error) {
	item, err := kr.keyring.Get(serverURL)
	if err != nil {
		logger.WithField("err", err).Error("keychain Get returned error")
		return "", "", credentials.ErrCredentialsNotFound
	}
	var creds credentials.Credentials
	if err = json.Unmarshal(item.Data, &creds); err != nil {
		logger.WithField("err", err).Error("stored credential malformed")
		return "", "", credentials.ErrCredentialsNotFound
	}

	return creds.Username, creds.Secret, nil
}

func (KeyringHelper) SupportsCredentialStorage() bool {
	return true
}
