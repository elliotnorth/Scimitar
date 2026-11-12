package core

type CatalogEntry struct {
	ID      string
	Name    string
	Aliases []string
}

var DefaultCatalog = []CatalogEntry{
	{ID: "nextcloud", Name: "Nextcloud", Aliases: []string{"nextcloud"}},
	{ID: "immich", Name: "Immich", Aliases: []string{"immich"}},
	{ID: "vaultwarden", Name: "Vaultwarden", Aliases: []string{"vaultwarden", "bitwarden"}},
	{ID: "jellyfin", Name: "Jellyfin", Aliases: []string{"jellyfin"}},
	{ID: "home-assistant", Name: "Home Assistant", Aliases: []string{"home-assistant", "hass"}},
	{ID: "pi-hole", Name: "Pi-hole", Aliases: []string{"pi-hole"}},
	{ID: "adguard-home", Name: "AdGuard Home", Aliases: []string{"adguard-home", "adguard"}},
	{ID: "gitea", Name: "Gitea", Aliases: []string{"gitea"}},
	{ID: "paperless-ngx", Name: "Paperless-ngx", Aliases: []string{"paperless-ngx", "paperless"}},
}
