package main

type RecordResponse struct {
	Result     []RecordResult `json:"result"`
	ResultInfo ResultInfo     `json:"result_info"`
	Success    bool           `json:"success"`
	Errors     []interface{}  `json:"errors"`
	Messages   []interface{}  `json:"messages"`
}

type ZoneResponse struct {
	Result     []ZoneResult  `json:"result"`
	ResultInfo ResultInfo    `json:"result_info"`
	Success    bool          `json:"success"`
	Errors     []interface{} `json:"errors"`
	Messages   []interface{} `json:"messages"`
}

type ZoneResult struct {
	ID                  string      `json:"id"`
	Name                string      `json:"name"`
	Status              string      `json:"status"`
	Paused              bool        `json:"paused"`
	Type                string      `json:"type"`
	DevelopmentMode     int64       `json:"development_mode"`
	NameServers         []string    `json:"name_servers"`
	OriginalNameServers []string    `json:"original_name_servers"`
	OriginalRegistrar   interface{} `json:"original_registrar"`
	OriginalDnshost     interface{} `json:"original_dnshost"`
	ModifiedOn          string      `json:"modified_on"`
	CreatedOn           string      `json:"created_on"`
	ActivatedOn         string      `json:"activated_on"`
	Meta                Meta        `json:"meta"`
	Owner               Owner       `json:"owner"`
	Account             Account     `json:"account"`
	Permissions         []string    `json:"permissions"`
	Plan                Plan        `json:"plan"`
}

type RecordResult struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	Proxiable  bool   `json:"proxiable"`
	Proxied    bool   `json:"proxied"`
	TTL        int64  `json:"ttl"`
	Locked     bool   `json:"locked"`
	ZoneID     string `json:"zone_id"`
	ZoneName   string `json:"zone_name"`
	ModifiedOn string `json:"modified_on"`
	CreatedOn  string `json:"created_on"`
	Meta       Meta   `json:"meta"`
}

type Meta struct {
	AutoAdded           bool `json:"auto_added"`
	ManagedByApps       bool `json:"managed_by_apps"`
	ManagedByArgoTunnel bool `json:"managed_by_argo_tunnel"`
}

type ResultInfo struct {
	Page       int64 `json:"page"`
	PerPage    int64 `json:"per_page"`
	TotalPages int64 `json:"total_pages"`
	Count      int64 `json:"count"`
	TotalCount int64 `json:"total_count"`
}

type Account struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Owner struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Email string `json:"email"`
}

type Plan struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Price             int64  `json:"price"`
	Currency          string `json:"currency"`
	Frequency         string `json:"frequency"`
	IsSubscribed      bool   `json:"is_subscribed"`
	CanSubscribe      bool   `json:"can_subscribe"`
	LegacyID          string `json:"legacy_id"`
	LegacyDiscount    bool   `json:"legacy_discount"`
	ExternallyManaged bool   `json:"externally_managed"`
}
