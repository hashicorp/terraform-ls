package algolia

import "context"

var credentialsKey struct{}

type Credentials struct {
	AppID  string
	APIKey string
}

func WithCredentials(ctx context.Context, appID, apiKey string) context.Context {
	return context.WithValue(ctx, credentialsKey, Credentials{
		AppID:  appID,
		APIKey: apiKey,
	})
}

func CredentialsFromContext(ctx context.Context) (Credentials, bool) {
	credentials, ok := ctx.Value(credentialsKey).(Credentials)
	return credentials, ok
}
