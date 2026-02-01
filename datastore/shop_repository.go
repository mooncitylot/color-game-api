package datastore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/color-game/api/models"
)

// ShopRepository defines the interface for shop-related database operations
type ShopRepository interface {
	// Shop Items
	CreateItem(item models.ShopItem) (models.ShopItem, error)
	GetItem(itemID string) (models.ShopItem, error)
	GetAllItems() ([]models.ShopItem, error)
	GetItemsByType(itemType string) ([]models.ShopItem, error)
	GetActiveItems() ([]models.ShopItem, error)
	UpdateItem(itemID string, updates models.UpdateShopItemRequest) (models.ShopItem, error)
	DeactivateItem(itemID string) error

	// User Inventory
	GetUserInventory(userID string) ([]models.UserInventoryWithItem, error)
	GetInventoryItem(inventoryID int) (models.UserInventoryItem, error)
	GetUserInventoryItem(userID string, itemID string) (models.UserInventoryItem, error)
	AddItemToInventory(userID string, itemID string, quantity int, expiresAt *time.Time) error
	UpdateInventoryQuantity(inventoryID int, quantity int) error
	EquipItem(inventoryID int, equip bool) error
	GetEquippedItems(userID string) ([]models.UserInventoryWithItem, error)
	UseItem(inventoryID int) error
	DeleteInventoryItem(inventoryID int) error

	// Purchases
	CreatePurchase(purchase models.PurchaseRecord) error
	GetUserPurchaseHistory(userID string) ([]models.PurchaseRecordWithItem, error)
	GetPurchasesByItem(itemID string) ([]models.PurchaseRecord, error)
}

// ShopDatabase implements ShopRepository
type ShopDatabase struct {
	database *sql.DB
}

// NewShopDatabase creates a new shop database instance
func NewShopDatabase(db *sql.DB) (ShopDatabase, error) {
	return ShopDatabase{database: db}, nil
}

// ============= SHOP ITEMS =============

// CreateItem creates a new shop item
func (sd ShopDatabase) CreateItem(item models.ShopItem) (models.ShopItem, error) {
	query := `
		INSERT INTO shop_items (
			item_id, item_type, name, description, credit_cost, rarity,
			metadata, is_active, is_limited_edition, stock_quantity,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING item_id, item_type, name, description, credit_cost, rarity,
			metadata, is_active, is_limited_edition, stock_quantity,
			created_at, updated_at`

	row := sd.database.QueryRow(
		query,
		item.ItemID,
		item.ItemType,
		item.Name,
		item.Description,
		item.CreditCost,
		item.Rarity,
		item.Metadata,
		item.IsActive,
		item.IsLimitedEdition,
		item.StockQuantity,
		item.CreatedAt,
		item.UpdatedAt,
	)

	var created models.ShopItem
	err := row.Scan(
		&created.ItemID,
		&created.ItemType,
		&created.Name,
		&created.Description,
		&created.CreditCost,
		&created.Rarity,
		&created.Metadata,
		&created.IsActive,
		&created.IsLimitedEdition,
		&created.StockQuantity,
		&created.CreatedAt,
		&created.UpdatedAt,
	)

	if err != nil {
		return models.ShopItem{}, fmt.Errorf("failed to create item: %v", err)
	}

	return created, nil
}

// GetItem retrieves a single shop item by ID
func (sd ShopDatabase) GetItem(itemID string) (models.ShopItem, error) {
	query := `
		SELECT item_id, item_type, name, description, credit_cost, rarity,
			metadata, is_active, is_limited_edition, stock_quantity,
			created_at, updated_at
		FROM shop_items
		WHERE item_id = $1`

	var item models.ShopItem
	err := sd.database.QueryRow(query, itemID).Scan(
		&item.ItemID,
		&item.ItemType,
		&item.Name,
		&item.Description,
		&item.CreditCost,
		&item.Rarity,
		&item.Metadata,
		&item.IsActive,
		&item.IsLimitedEdition,
		&item.StockQuantity,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return models.ShopItem{}, NoRowsError{true, err}
	}
	if err != nil {
		return models.ShopItem{}, fmt.Errorf("failed to get item: %v", err)
	}

	return item, nil
}

// GetAllItems retrieves all shop items
func (sd ShopDatabase) GetAllItems() ([]models.ShopItem, error) {
	query := `
		SELECT item_id, item_type, name, description, credit_cost, rarity,
			metadata, is_active, is_limited_edition, stock_quantity,
			created_at, updated_at
		FROM shop_items
		ORDER BY created_at DESC`

	return sd.queryItems(query)
}

// GetItemsByType retrieves shop items by type
func (sd ShopDatabase) GetItemsByType(itemType string) ([]models.ShopItem, error) {
	query := `
		SELECT item_id, item_type, name, description, credit_cost, rarity,
			metadata, is_active, is_limited_edition, stock_quantity,
			created_at, updated_at
		FROM shop_items
		WHERE item_type = $1
		ORDER BY created_at DESC`

	rows, err := sd.database.Query(query, itemType)
	if err != nil {
		return nil, fmt.Errorf("failed to query items by type: %v", err)
	}
	defer rows.Close()

	return sd.scanItems(rows)
}

// GetActiveItems retrieves all active shop items
func (sd ShopDatabase) GetActiveItems() ([]models.ShopItem, error) {
	query := `
		SELECT item_id, item_type, name, description, credit_cost, rarity,
			metadata, is_active, is_limited_edition, stock_quantity,
			created_at, updated_at
		FROM shop_items
		WHERE is_active = true
		ORDER BY rarity DESC, created_at DESC`

	return sd.queryItems(query)
}

// UpdateItem updates a shop item
func (sd ShopDatabase) UpdateItem(itemID string, updates models.UpdateShopItemRequest) (models.ShopItem, error) {
	// Start building dynamic update query
	query := "UPDATE shop_items SET updated_at = $1"
	args := []interface{}{time.Now()}
	argIndex := 2

	if updates.Name != nil {
		query += fmt.Sprintf(", name = $%d", argIndex)
		args = append(args, *updates.Name)
		argIndex++
	}
	if updates.Description != nil {
		query += fmt.Sprintf(", description = $%d", argIndex)
		args = append(args, *updates.Description)
		argIndex++
	}
	if updates.CreditCost != nil {
		query += fmt.Sprintf(", credit_cost = $%d", argIndex)
		args = append(args, *updates.CreditCost)
		argIndex++
	}
	if updates.Rarity != nil {
		query += fmt.Sprintf(", rarity = $%d", argIndex)
		args = append(args, *updates.Rarity)
		argIndex++
	}
	if updates.Metadata != nil {
		query += fmt.Sprintf(", metadata = $%d", argIndex)
		args = append(args, updates.Metadata)
		argIndex++
	}
	if updates.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", argIndex)
		args = append(args, *updates.IsActive)
		argIndex++
	}
	if updates.IsLimitedEdition != nil {
		query += fmt.Sprintf(", is_limited_edition = $%d", argIndex)
		args = append(args, *updates.IsLimitedEdition)
		argIndex++
	}
	if updates.StockQuantity != nil {
		query += fmt.Sprintf(", stock_quantity = $%d", argIndex)
		args = append(args, updates.StockQuantity)
		argIndex++
	}

	query += fmt.Sprintf(" WHERE item_id = $%d RETURNING item_id, item_type, name, description, credit_cost, rarity, metadata, is_active, is_limited_edition, stock_quantity, created_at, updated_at", argIndex)
	args = append(args, itemID)

	var item models.ShopItem
	err := sd.database.QueryRow(query, args...).Scan(
		&item.ItemID,
		&item.ItemType,
		&item.Name,
		&item.Description,
		&item.CreditCost,
		&item.Rarity,
		&item.Metadata,
		&item.IsActive,
		&item.IsLimitedEdition,
		&item.StockQuantity,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		return models.ShopItem{}, fmt.Errorf("failed to update item: %v", err)
	}

	return item, nil
}

// DeactivateItem soft deletes a shop item by setting is_active to false
func (sd ShopDatabase) DeactivateItem(itemID string) error {
	query := `UPDATE shop_items SET is_active = false, updated_at = $1 WHERE item_id = $2`
	_, err := sd.database.Exec(query, time.Now(), itemID)
	if err != nil {
		return fmt.Errorf("failed to deactivate item: %v", err)
	}
	return nil
}

// ============= USER INVENTORY =============

// GetUserInventory retrieves all items in a user's inventory
func (sd ShopDatabase) GetUserInventory(userID string) ([]models.UserInventoryWithItem, error) {
	query := `
		SELECT 
			ui.inventory_id, ui.user_id, ui.item_id, ui.quantity,
			ui.is_equipped, ui.acquired_at, ui.expires_at, ui.used_count,
			si.item_id, si.item_type, si.name, si.description, si.credit_cost,
			si.rarity, si.metadata, si.is_active, si.is_limited_edition,
			si.stock_quantity, si.created_at, si.updated_at
		FROM user_inventory ui
		JOIN shop_items si ON ui.item_id = si.item_id
		WHERE ui.user_id = $1
		ORDER BY ui.acquired_at DESC`

	rows, err := sd.database.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user inventory: %v", err)
	}
	defer rows.Close()

	var inventory []models.UserInventoryWithItem
	for rows.Next() {
		var item models.UserInventoryWithItem
		err := rows.Scan(
			&item.InventoryID,
			&item.UserID,
			&item.ItemID,
			&item.Quantity,
			&item.IsEquipped,
			&item.AcquiredAt,
			&item.ExpiresAt,
			&item.UsedCount,
			&item.ShopItem.ItemID,
			&item.ShopItem.ItemType,
			&item.ShopItem.Name,
			&item.ShopItem.Description,
			&item.ShopItem.CreditCost,
			&item.ShopItem.Rarity,
			&item.ShopItem.Metadata,
			&item.ShopItem.IsActive,
			&item.ShopItem.IsLimitedEdition,
			&item.ShopItem.StockQuantity,
			&item.ShopItem.CreatedAt,
			&item.ShopItem.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory item: %v", err)
		}
		inventory = append(inventory, item)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating inventory: %v", rows.Err())
	}

	return inventory, nil
}

// GetInventoryItem retrieves a single inventory item by ID
func (sd ShopDatabase) GetInventoryItem(inventoryID int) (models.UserInventoryItem, error) {
	query := `
		SELECT inventory_id, user_id, item_id, quantity, is_equipped,
			acquired_at, expires_at, used_count
		FROM user_inventory
		WHERE inventory_id = $1`

	var item models.UserInventoryItem
	err := sd.database.QueryRow(query, inventoryID).Scan(
		&item.InventoryID,
		&item.UserID,
		&item.ItemID,
		&item.Quantity,
		&item.IsEquipped,
		&item.AcquiredAt,
		&item.ExpiresAt,
		&item.UsedCount,
	)

	if err == sql.ErrNoRows {
		return models.UserInventoryItem{}, NoRowsError{true, err}
	}
	if err != nil {
		return models.UserInventoryItem{}, fmt.Errorf("failed to get inventory item: %v", err)
	}

	return item, nil
}

// GetUserInventoryItem retrieves a specific item from user's inventory
func (sd ShopDatabase) GetUserInventoryItem(userID string, itemID string) (models.UserInventoryItem, error) {
	query := `
		SELECT inventory_id, user_id, item_id, quantity, is_equipped,
			acquired_at, expires_at, used_count
		FROM user_inventory
		WHERE user_id = $1 AND item_id = $2`

	var item models.UserInventoryItem
	err := sd.database.QueryRow(query, userID, itemID).Scan(
		&item.InventoryID,
		&item.UserID,
		&item.ItemID,
		&item.Quantity,
		&item.IsEquipped,
		&item.AcquiredAt,
		&item.ExpiresAt,
		&item.UsedCount,
	)

	if err == sql.ErrNoRows {
		return models.UserInventoryItem{}, NoRowsError{true, err}
	}
	if err != nil {
		return models.UserInventoryItem{}, fmt.Errorf("failed to get user inventory item: %v", err)
	}

	return item, nil
}

// AddItemToInventory adds an item to user's inventory or updates quantity if exists
func (sd ShopDatabase) AddItemToInventory(userID string, itemID string, quantity int, expiresAt *time.Time) error {
	query := `
		INSERT INTO user_inventory (user_id, item_id, quantity, expires_at, acquired_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, item_id)
		DO UPDATE SET quantity = user_inventory.quantity + $3`

	_, err := sd.database.Exec(query, userID, itemID, quantity, expiresAt, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add item to inventory: %v", err)
	}

	return nil
}

// UpdateInventoryQuantity updates the quantity of an inventory item
func (sd ShopDatabase) UpdateInventoryQuantity(inventoryID int, quantity int) error {
	query := `UPDATE user_inventory SET quantity = $1 WHERE inventory_id = $2`
	_, err := sd.database.Exec(query, quantity, inventoryID)
	if err != nil {
		return fmt.Errorf("failed to update inventory quantity: %v", err)
	}
	return nil
}

// EquipItem equips or unequips an item
func (sd ShopDatabase) EquipItem(inventoryID int, equip bool) error {
	query := `UPDATE user_inventory SET is_equipped = $1 WHERE inventory_id = $2`
	_, err := sd.database.Exec(query, equip, inventoryID)
	if err != nil {
		return fmt.Errorf("failed to equip item: %v", err)
	}
	return nil
}

// GetEquippedItems retrieves all equipped items for a user
func (sd ShopDatabase) GetEquippedItems(userID string) ([]models.UserInventoryWithItem, error) {
	query := `
		SELECT 
			ui.inventory_id, ui.user_id, ui.item_id, ui.quantity,
			ui.is_equipped, ui.acquired_at, ui.expires_at, ui.used_count,
			si.item_id, si.item_type, si.name, si.description, si.credit_cost,
			si.rarity, si.metadata, si.is_active, si.is_limited_edition,
			si.stock_quantity, si.created_at, si.updated_at
		FROM user_inventory ui
		JOIN shop_items si ON ui.item_id = si.item_id
		WHERE ui.user_id = $1 AND ui.is_equipped = true`

	rows, err := sd.database.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipped items: %v", err)
	}
	defer rows.Close()

	var items []models.UserInventoryWithItem
	for rows.Next() {
		var item models.UserInventoryWithItem
		err := rows.Scan(
			&item.InventoryID,
			&item.UserID,
			&item.ItemID,
			&item.Quantity,
			&item.IsEquipped,
			&item.AcquiredAt,
			&item.ExpiresAt,
			&item.UsedCount,
			&item.ShopItem.ItemID,
			&item.ShopItem.ItemType,
			&item.ShopItem.Name,
			&item.ShopItem.Description,
			&item.ShopItem.CreditCost,
			&item.ShopItem.Rarity,
			&item.ShopItem.Metadata,
			&item.ShopItem.IsActive,
			&item.ShopItem.IsLimitedEdition,
			&item.ShopItem.StockQuantity,
			&item.ShopItem.CreatedAt,
			&item.ShopItem.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan equipped item: %v", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// UseItem increments the used_count for a consumable item
func (sd ShopDatabase) UseItem(inventoryID int) error {
	query := `
		UPDATE user_inventory 
		SET used_count = used_count + 1, quantity = quantity - 1
		WHERE inventory_id = $1 AND quantity > 0`

	result, err := sd.database.Exec(query, inventoryID)
	if err != nil {
		return fmt.Errorf("failed to use item: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("item not found or out of stock")
	}

	return nil
}

// DeleteInventoryItem removes an item from inventory
func (sd ShopDatabase) DeleteInventoryItem(inventoryID int) error {
	query := `DELETE FROM user_inventory WHERE inventory_id = $1`
	_, err := sd.database.Exec(query, inventoryID)
	if err != nil {
		return fmt.Errorf("failed to delete inventory item: %v", err)
	}
	return nil
}

// ============= PURCHASES =============

// CreatePurchase records a purchase transaction
func (sd ShopDatabase) CreatePurchase(purchase models.PurchaseRecord) error {
	query := `
		INSERT INTO purchase_history (purchase_id, user_id, item_id, quantity, credits_spent, purchased_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := sd.database.Exec(
		query,
		purchase.PurchaseID,
		purchase.UserID,
		purchase.ItemID,
		purchase.Quantity,
		purchase.CreditsSpent,
		purchase.PurchasedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create purchase record: %v", err)
	}

	return nil
}

// GetUserPurchaseHistory retrieves purchase history for a user
func (sd ShopDatabase) GetUserPurchaseHistory(userID string) ([]models.PurchaseRecordWithItem, error) {
	query := `
		SELECT 
			ph.purchase_id, ph.user_id, ph.item_id, ph.quantity,
			ph.credits_spent, ph.purchased_at,
			si.item_id, si.item_type, si.name, si.description, si.credit_cost,
			si.rarity, si.metadata, si.is_active, si.is_limited_edition,
			si.stock_quantity, si.created_at, si.updated_at
		FROM purchase_history ph
		JOIN shop_items si ON ph.item_id = si.item_id
		WHERE ph.user_id = $1
		ORDER BY ph.purchased_at DESC`

	rows, err := sd.database.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get purchase history: %v", err)
	}
	defer rows.Close()

	var purchases []models.PurchaseRecordWithItem
	for rows.Next() {
		var purchase models.PurchaseRecordWithItem
		err := rows.Scan(
			&purchase.PurchaseID,
			&purchase.UserID,
			&purchase.ItemID,
			&purchase.Quantity,
			&purchase.CreditsSpent,
			&purchase.PurchasedAt,
			&purchase.ShopItem.ItemID,
			&purchase.ShopItem.ItemType,
			&purchase.ShopItem.Name,
			&purchase.ShopItem.Description,
			&purchase.ShopItem.CreditCost,
			&purchase.ShopItem.Rarity,
			&purchase.ShopItem.Metadata,
			&purchase.ShopItem.IsActive,
			&purchase.ShopItem.IsLimitedEdition,
			&purchase.ShopItem.StockQuantity,
			&purchase.ShopItem.CreatedAt,
			&purchase.ShopItem.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan purchase record: %v", err)
		}
		purchases = append(purchases, purchase)
	}

	return purchases, nil
}

// GetPurchasesByItem retrieves all purchases of a specific item
func (sd ShopDatabase) GetPurchasesByItem(itemID string) ([]models.PurchaseRecord, error) {
	query := `
		SELECT purchase_id, user_id, item_id, quantity, credits_spent, purchased_at
		FROM purchase_history
		WHERE item_id = $1
		ORDER BY purchased_at DESC`

	rows, err := sd.database.Query(query, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get purchases by item: %v", err)
	}
	defer rows.Close()

	var purchases []models.PurchaseRecord
	for rows.Next() {
		var purchase models.PurchaseRecord
		err := rows.Scan(
			&purchase.PurchaseID,
			&purchase.UserID,
			&purchase.ItemID,
			&purchase.Quantity,
			&purchase.CreditsSpent,
			&purchase.PurchasedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan purchase: %v", err)
		}
		purchases = append(purchases, purchase)
	}

	return purchases, nil
}

// ============= HELPER FUNCTIONS =============

// queryItems executes a query and returns shop items
func (sd ShopDatabase) queryItems(query string, args ...interface{}) ([]models.ShopItem, error) {
	rows, err := sd.database.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query items: %v", err)
	}
	defer rows.Close()

	return sd.scanItems(rows)
}

// scanItems scans rows into ShopItem slice
func (sd ShopDatabase) scanItems(rows *sql.Rows) ([]models.ShopItem, error) {
	var items []models.ShopItem
	for rows.Next() {
		var item models.ShopItem
		var metadataBytes []byte

		err := rows.Scan(
			&item.ItemID,
			&item.ItemType,
			&item.Name,
			&item.Description,
			&item.CreditCost,
			&item.Rarity,
			&metadataBytes,
			&item.IsActive,
			&item.IsLimitedEdition,
			&item.StockQuantity,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %v", err)
		}

		// Convert metadata bytes to json.RawMessage
		if len(metadataBytes) > 0 {
			item.Metadata = json.RawMessage(metadataBytes)
		}

		items = append(items, item)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating items: %v", rows.Err())
	}

	return items, nil
}
