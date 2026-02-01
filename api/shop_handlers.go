package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/color-game/api/datastore"
	"github.com/color-game/api/models"
)

// ============= SHOP ITEMS =============

// GET /v1/shop/items - Get all active shop items
func (app *Application) getShopItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check for item type filter
	itemType := r.URL.Query().Get("type")

	var items []models.ShopItem
	var err error

	if itemType != "" {
		items, err = app.ShopRepo.GetItemsByType(itemType)
	} else {
		items, err = app.ShopRepo.GetActiveItems()
	}

	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(items)
}

// GET /v1/shop/items/:id - Get a specific shop item
func (app *Application) getShopItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	itemID := r.URL.Query().Get("id")
	if itemID == "" {
		app.badRequest(w, r, errors.New("item ID is required"))
		return
	}

	item, err := app.ShopRepo.GetItem(itemID)
	if err != nil {
		if _, ok := err.(datastore.NoRowsError); ok {
			http.Error(w, "Item not found", http.StatusNotFound)
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(item)
}

// POST /v1/shop/purchase - Purchase an item
func (app *Application) purchaseItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.requirePostMethod(w, r, ErrPOST)
		return
	}

	// Get current user from token
	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	// Parse purchase request
	var purchaseReq models.PurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&purchaseReq); err != nil {
		app.badJSONRequest(w, r, err)
		return
	}

	// Validate quantity
	if purchaseReq.Quantity <= 0 {
		app.badRequest(w, r, errors.New("quantity must be greater than 0"))
		return
	}

	// Get the item
	item, err := app.ShopRepo.GetItem(purchaseReq.ItemID)
	if err != nil {
		if _, ok := err.(datastore.NoRowsError); ok {
			http.Error(w, "Item not found", http.StatusNotFound)
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	// Check if item is active
	if !item.IsActive {
		app.badRequest(w, r, errors.New("item is not available for purchase"))
		return
	}

	// Check stock availability
	if item.StockQuantity != nil && *item.StockQuantity < purchaseReq.Quantity {
		app.badRequest(w, r, errors.New("insufficient stock available"))
		return
	}

	// Calculate total cost
	totalCost := item.CreditCost * purchaseReq.Quantity

	// Check if user has enough credits
	if user.Credits < totalCost {
		app.badRequest(w, r, fmt.Errorf("insufficient credits. Need %d, have %d", totalCost, user.Credits))
		return
	}

	// Start transaction logic
	// 1. Deduct credits from user
	user.Credits -= totalCost
	_, err = app.UserRepo.Update(user)
	if err != nil {
		app.internalServerError(w, r, fmt.Errorf("failed to deduct credits: %v", err))
		return
	}

	// 2. Add item to user's inventory
	err = app.ShopRepo.AddItemToInventory(user.UserID, item.ItemID, purchaseReq.Quantity, nil)
	if err != nil {
		// Rollback: Add credits back
		user.Credits += totalCost
		app.UserRepo.Update(user)
		app.internalServerError(w, r, fmt.Errorf("failed to add item to inventory: %v", err))
		return
	}

	// 3. Update stock if limited edition
	if item.StockQuantity != nil {
		newStock := *item.StockQuantity - purchaseReq.Quantity
		updates := models.UpdateShopItemRequest{
			StockQuantity: &newStock,
		}
		_, err = app.ShopRepo.UpdateItem(item.ItemID, updates)
		if err != nil {
			// Note: This is a non-critical error, log but don't fail the purchase
			fmt.Printf("Warning: Failed to update stock for item %s: %v\n", item.ItemID, err)
		}
	}

	// 4. Record the purchase
	purchase := models.PurchaseRecord{
		PurchaseID:   models.GeneratePurchaseID(),
		UserID:       user.UserID,
		ItemID:       item.ItemID,
		Quantity:     purchaseReq.Quantity,
		CreditsSpent: totalCost,
		PurchasedAt:  time.Now(),
	}

	err = app.ShopRepo.CreatePurchase(purchase)
	if err != nil {
		// Non-critical error, log but don't fail
		fmt.Printf("Warning: Failed to record purchase: %v\n", err)
	}

	// Build response
	response := map[string]interface{}{
		"message":          "Purchase successful",
		"item":             item,
		"quantity":         purchaseReq.Quantity,
		"creditsSpent":     totalCost,
		"creditsRemaining": user.Credits,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ============= INVENTORY =============

// GET /v1/inventory - Get user's inventory
func (app *Application) getUserInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current user from token
	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	// Get inventory
	inventory, err := app.ShopRepo.GetUserInventory(user.UserID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(inventory)
}

// GET /v1/inventory/equipped - Get user's equipped items
func (app *Application) getEquippedItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current user from token
	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	// Get equipped items
	equippedItems, err := app.ShopRepo.GetEquippedItems(user.UserID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(equippedItems)
}

// PUT /v1/inventory/equip - Equip/unequip an item
func (app *Application) equipItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		app.requirePutMethod(w, r, ErrPUT)
		return
	}

	// Get current user from token
	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	// Parse request
	var equipReq models.EquipItemRequest
	if err := json.NewDecoder(r.Body).Decode(&equipReq); err != nil {
		app.badJSONRequest(w, r, err)
		return
	}

	// Get inventory item to verify ownership
	inventoryItem, err := app.ShopRepo.GetInventoryItem(equipReq.InventoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Inventory item not found", http.StatusNotFound)
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	// Verify the item belongs to the user
	if inventoryItem.UserID != user.UserID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Equip or unequip
	err = app.ShopRepo.EquipItem(equipReq.InventoryID, equipReq.Equip)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	message := "Item unequipped"
	if equipReq.Equip {
		message = "Item equipped"
	}

	response := map[string]interface{}{
		"message":     message,
		"inventoryId": equipReq.InventoryID,
		"equipped":    equipReq.Equip,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// POST /v1/inventory/use - Use a consumable item
func (app *Application) useItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.requirePostMethod(w, r, ErrPOST)
		return
	}

	// Get current user from token
	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	// Parse request
	var useReq models.UseItemRequest
	if err := json.NewDecoder(r.Body).Decode(&useReq); err != nil {
		app.badJSONRequest(w, r, err)
		return
	}

	// Get inventory item to verify ownership
	inventoryItem, err := app.ShopRepo.GetInventoryItem(useReq.InventoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Inventory item not found", http.StatusNotFound)
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	// Verify the item belongs to the user
	if inventoryItem.UserID != user.UserID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Check if item is available (not expired, has quantity)
	if inventoryItem.Quantity <= 0 {
		app.badRequest(w, r, errors.New("item out of stock"))
		return
	}

	if inventoryItem.ExpiresAt != nil && inventoryItem.ExpiresAt.Before(time.Now()) {
		app.badRequest(w, r, errors.New("item has expired"))
		return
	}

	// Load the underlying shop item to determine effects
	shopItem, err := app.ShopRepo.GetItem(inventoryItem.ItemID)
	if err != nil {
		app.internalServerError(w, r, fmt.Errorf("failed to load item %s: %v", inventoryItem.ItemID, err))
		return
	}

	var effectMetadata map[string]any
	if len(shopItem.Metadata) > 0 {
		effectMetadata = map[string]any{}
		if err := json.Unmarshal(shopItem.Metadata, &effectMetadata); err != nil {
			app.internalServerError(w, r, fmt.Errorf("failed to parse item metadata: %v", err))
			return
		}
	}

	// Use the item
	err = app.ShopRepo.UseItem(useReq.InventoryID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// Get updated inventory item
	updatedItem, err := app.ShopRepo.GetInventoryItem(useReq.InventoryID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	response := models.UseItemResponse{
		Message:       "Item used successfully",
		InventoryID:   useReq.InventoryID,
		QuantityLeft:  updatedItem.Quantity,
		UsedCount:     updatedItem.UsedCount,
		Item:          &shopItem,
		InventoryItem: &updatedItem,
	}

	// Apply effect logic for consumables like Extra Scan
	if len(effectMetadata) > 0 {
		response.EffectMetadata = effectMetadata

		if effectType, _ := effectMetadata["effect_type"].(string); effectType == "extra_attempt" {
			extraAttempts := 1
			if raw, ok := effectMetadata["extra_attempts"]; ok {
				switch v := raw.(type) {
				case float64:
					if attemptInt := int(v); attemptInt > 0 {
						extraAttempts = attemptInt
					}
				case int:
					if v > 0 {
						extraAttempts = v
					}
				case string:
					if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
						extraAttempts = parsed
					}
				}
			}

			now := time.Now()
			normalizedDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			modifier, err := app.DailyScoreRepo.SetDailyAttemptModifier(user.UserID, normalizedDate, extraAttempts)
			if err != nil {
				app.internalServerError(w, r, fmt.Errorf("failed to apply extra attempts: %v", err))
				return
			}

			if response.EffectMetadata == nil {
				response.EffectMetadata = map[string]any{}
			}

			response.EffectMetadata["extra_attempts_applied"] = extraAttempts
			response.EffectMetadata["total_extra_attempts"] = modifier.ExtraAttempts
			response.EffectMetadata["max_attempts"] = 5 + modifier.ExtraAttempts
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ============= PURCHASE HISTORY =============

// GET /v1/shop/purchases - Get user's purchase history
func (app *Application) getPurchaseHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current user from token
	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	// Get purchase history
	purchases, err := app.ShopRepo.GetUserPurchaseHistory(user.UserID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(purchases)
}

// ============= ADMIN ENDPOINTS =============

// POST /v1/admin/shop/items - Create a new shop item (Admin only)
func (app *Application) createShopItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.requirePostMethod(w, r, ErrPOST)
		return
	}

	// Parse request
	var createReq models.CreateShopItemRequest
	if err := json.NewDecoder(r.Body).Decode(&createReq); err != nil {
		app.badJSONRequest(w, r, err)
		return
	}

	// Validate required fields
	if createReq.Name == "" || createReq.ItemType == "" {
		app.badRequest(w, r, errors.New("name and itemType are required"))
		return
	}

	if createReq.CreditCost < 0 {
		app.badRequest(w, r, errors.New("creditCost must be non-negative"))
		return
	}

	// Create shop item
	newItem := models.NewShopItem(createReq)

	// Save to database
	createdItem, err := app.ShopRepo.CreateItem(newItem)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdItem)
}

// GET /v1/admin/shop/items - Get all shop items including inactive (Admin only)
func (app *Application) getAllShopItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	items, err := app.ShopRepo.GetAllItems()
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(items)
}

// PUT /v1/admin/shop/items - Update a shop item (Admin only)
func (app *Application) updateShopItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		app.requirePutMethod(w, r, ErrPUT)
		return
	}

	itemID := r.URL.Query().Get("id")
	if itemID == "" {
		app.badRequest(w, r, errors.New("item ID is required"))
		return
	}

	// Parse update request
	var updateReq models.UpdateShopItemRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		app.badJSONRequest(w, r, err)
		return
	}

	// Update the item
	updatedItem, err := app.ShopRepo.UpdateItem(itemID, updateReq)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedItem)
}

// DELETE /v1/admin/shop/items - Deactivate a shop item (Admin only)
func (app *Application) deactivateShopItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	itemID := r.URL.Query().Get("id")
	if itemID == "" {
		app.badRequest(w, r, errors.New("item ID is required"))
		return
	}

	err := app.ShopRepo.DeactivateItem(itemID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	response := map[string]interface{}{
		"message": "Item deactivated successfully",
		"itemId":  itemID,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// POST /v1/admin/users/credits - Add credits to a user (Admin only)
func (app *Application) addUserCredits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.requirePostMethod(w, r, ErrPOST)
		return
	}

	// Parse request
	var req struct {
		UserID  string `json:"userId"`
		Credits int    `json:"credits"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.badJSONRequest(w, r, err)
		return
	}

	if req.UserID == "" {
		app.badRequest(w, r, errors.New("userId is required"))
		return
	}

	if req.Credits <= 0 {
		app.badRequest(w, r, errors.New("credits must be positive"))
		return
	}

	// Get user
	user, err := app.UserRepo.Get(req.UserID)
	if err != nil {
		if _, ok := err.(datastore.NoRowsError); ok {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	// Add credits
	user.Credits += req.Credits
	updatedUser, err := app.UserRepo.Update(user)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	response := map[string]interface{}{
		"message":      fmt.Sprintf("Added %d credits to user", req.Credits),
		"userId":       user.UserID,
		"username":     user.Username,
		"totalCredits": updatedUser.Credits,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GET /v1/admin/shop/purchases - Get all purchases or by item (Admin only)
func (app *Application) getAdminPurchases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	itemID := r.URL.Query().Get("itemId")

	if itemID != "" {
		// Get purchases for specific item
		purchases, err := app.ShopRepo.GetPurchasesByItem(itemID)
		if err != nil {
			app.internalServerError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(purchases)
		return
	}

	// For getting all purchases, we'd need to add a new method to the repository
	// For now, return an error suggesting to use itemId filter
	app.badRequest(w, r, errors.New("itemId parameter is required"))
}

// Helper function to parse inventory ID from query params
func parseInventoryID(r *http.Request) (int, error) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		return 0, errors.New("inventory ID is required")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, errors.New("invalid inventory ID")
	}

	return id, nil
}
