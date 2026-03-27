package handlers

type AdminLoginRequest struct {
	Password string `json:"password"`
}

type AdminLoginResponse struct {
	Token string `json:"token"`
}

type CreateProviderRequest struct {
	Name string `json:"name"`
}

type UpdateProviderRequest struct {
	APIKey string `json:"api_key"`
}

type CreateProviderResponse struct {
	UUID       string `json:"uuid"`
	APIKey     string `json:"api_key"`
	APIKeyHash string `json:"api_key_hash"`
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
}

type ProviderResponse struct {
	UUID       string `json:"uuid"`
	APIKeyHash string `json:"api_key_hash"`
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
}

type ProviderListResponse struct {
	Data []ProviderResponse `json:"data"`
	Meta PaginationMeta     `json:"meta"`
}

type CreateWalletResponse struct {
	UUID        string `json:"uuid"`
	Passphrase  string `json:"passphrase"`
	AccessToken string `json:"access_token"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type RegenerateWalletResponse struct {
	UUID        string `json:"uuid"`
	Passphrase  string `json:"passphrase"`
	AccessToken string `json:"access_token"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CreateCurrencyRequest struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type CurrencyAdminResponse struct {
	UUID        string  `json:"uuid"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

type CurrencyListAdminResponse struct {
	Data []CurrencyAdminResponse `json:"data"`
	Meta PaginationMeta          `json:"meta"`
}

type IssueCurrencyRequest struct {
	CurrencyID string  `json:"currency_id"`
	Amount     string  `json:"amount"`
	Reference  *string `json:"reference,omitempty"`
}

type IssueCurrencyResponse struct {
	Balance     *BalanceResponse    `json:"balance"`
	Transaction TransactionResponse `json:"transaction"`
	WalletUUID  string              `json:"wallet_uuid"`
}
