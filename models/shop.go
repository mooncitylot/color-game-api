package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Item types
const (
	ItemTypePowerup    = "powerup"
	ItemTypeBadge      = "badge"
	ItemTypeAvatarHat  = "avatar_hat"
	ItemTypeAvatarSkin = "avatar_skin"
)

// Item rarities
const (
	RarityCommon    = "common"
	RarityRare      = "rare"
	RarityEpic      = "epic"
	RarityLegendary = "legendary"
)

// ShopItem represents an item available for purchase in the shop
type ShopItem struct {
	ItemID           string          `json:"itemId" db:"item_id"`
	ItemType         string          `json:"itemType" db:"item_type"`
	Name             string          `json:"name" db:"name"`
	Description      string          `json:"description" db:"description"`
	CreditCost       int             `json:"creditCost" db:"credit_cost"`
	Rarity           string          `json:"rarity" db:"rarity"`
	Metadata         json.RawMessage `json:"metadata" db:"metadata"`
	IsActive         bool            `json:"isActive" db:"is_active"`
	IsLimitedEdition bool            `json:"isLimitedEdition" db:"is_limited_edition"`
	StockQuantity    *int            `json:"stockQuantity,omitempty" db:"stock_quantity"`
	CreatedAt        time.Time       `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time       `json:"updatedAt" db:"updated_at"`
}

// CreateShopItemRequest represents the request to create a new shop item
type CreateShopItemRequest struct {
	ItemType         string          `json:"itemType"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	CreditCost       int             `json:"creditCost"`
	Rarity           string          `json:"rarity"`
	Metadata         json.RawMessage `json:"metadata"`
	IsLimitedEdition bool            `json:"isLimitedEdition"`
	StockQuantity    *int            `json:"stockQuantity,omitempty"`
}

// UpdateShopItemRequest represents the request to update a shop item
type UpdateShopItemRequest struct {
	Name             *string         `json:"name,omitempty"`
	Description      *string         `json:"description,omitempty"`
	CreditCost       *int            `json:"creditCost,omitempty"`
	Rarity           *string         `json:"rarity,omitempty"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	IsActive         *bool           `json:"isActive,omitempty"`
	IsLimitedEdition *bool           `json:"isLimitedEdition,omitempty"`
	StockQuantity    *int            `json:"stockQuantity,omitempty"`
}

// UserInventoryItem represents an item owned by a user
type UserInventoryItem struct {
	InventoryID int        `json:"inventoryId" db:"inventory_id"`
	UserID      string     `json:"userId" db:"user_id"`
	ItemID      string     `json:"itemId" db:"item_id"`
	Quantity    int        `json:"quantity" db:"quantity"`
	IsEquipped  bool       `json:"isEquipped" db:"is_equipped"`
	AcquiredAt  time.Time  `json:"acquiredAt" db:"acquired_at"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty" db:"expires_at"`
	UsedCount   int        `json:"usedCount" db:"used_count"`
}

// UserInventoryWithItem represents inventory item with full shop item details
type UserInventoryWithItem struct {
	UserInventoryItem
	ShopItem ShopItem `json:"item"`
}

// PurchaseRequest represents a request to purchase an item
type PurchaseRequest struct {
	ItemID   string `json:"itemId"`
	Quantity int    `json:"quantity"`
}

// PurchaseRecord represents a purchase transaction
type PurchaseRecord struct {
	PurchaseID   string    `json:"purchaseId" db:"purchase_id"`
	UserID       string    `json:"userId" db:"user_id"`
	ItemID       string    `json:"itemId" db:"item_id"`
	Quantity     int       `json:"quantity" db:"quantity"`
	CreditsSpent int       `json:"creditsSpent" db:"credits_spent"`
	PurchasedAt  time.Time `json:"purchasedAt" db:"purchased_at"`
}

// PurchaseRecordWithItem represents purchase history with full item details
type PurchaseRecordWithItem struct {
	PurchaseRecord
	ShopItem ShopItem `json:"item"`
}

// EquipItemRequest represents a request to equip/unequip an item
type EquipItemRequest struct {
	InventoryID int  `json:"inventoryId"`
	Equip       bool `json:"equip"`
}

// UseItemRequest represents a request to use a consumable item
type UseItemRequest struct {
	InventoryID int `json:"inventoryId"`
}

// UseItemResponse describes the outcome of using an item
type UseItemResponse struct {
	Message        string             `json:"message"`
	InventoryID    int                `json:"inventoryId"`
	QuantityLeft   int                `json:"quantityLeft"`
	UsedCount      int                `json:"usedCount"`
	EffectMetadata map[string]any     `json:"effectMetadata,omitempty"`
	Item           *ShopItem          `json:"item,omitempty"`
	InventoryItem  *UserInventoryItem `json:"inventory,omitempty"`
}

// GenerateItemID creates a new unique ID for a shop item
func GenerateItemID() string {
	return uuid.New().String()
}

// GeneratePurchaseID creates a new unique ID for a purchase
func GeneratePurchaseID() string {
	return uuid.New().String()
}

// NewShopItem creates a new ShopItem from a CreateShopItemRequest
func NewShopItem(req CreateShopItemRequest) ShopItem {
	now := time.Now()
	return ShopItem{
		ItemID:           GenerateItemID(),
		ItemType:         req.ItemType,
		Name:             req.Name,
		Description:      req.Description,
		CreditCost:       req.CreditCost,
		Rarity:           req.Rarity,
		Metadata:         req.Metadata,
		IsActive:         true,
		IsLimitedEdition: req.IsLimitedEdition,
		StockQuantity:    req.StockQuantity,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}
