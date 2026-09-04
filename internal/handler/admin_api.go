package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"telegram-shop/internal/bot"
	"telegram-shop/internal/model"
	"telegram-shop/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AdminHandler handles admin REST API requests.
type AdminHandler struct {
	bot            *bot.Bot
	productService *service.ProductService
	orderService   *service.OrderService
	walletService  *service.WalletService
	userService    *service.UserService
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(
	bot *bot.Bot,
	productService *service.ProductService,
	orderService *service.OrderService,
	walletService *service.WalletService,
	userService *service.UserService,
) *AdminHandler {
	return &AdminHandler{
		bot:            bot,
		productService: productService,
		orderService:   orderService,
		walletService:  walletService,
		userService:    userService,
	}
}

// RegisterRoutes registers admin API routes on a chi router.
func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Get("/products", h.ListProducts)
	r.Post("/products", h.CreateProduct)
	r.Put("/products/{id}", h.UpdateProduct)
	r.Delete("/products/{id}", h.DeleteProduct)

	r.Get("/items", h.ListItems)
	r.Post("/items", h.CreateItem)
	r.Put("/items/{id}", h.UpdateItem)
	r.Delete("/items/{id}", h.DeleteItem)

	r.Get("/product-links", h.ListLinks)
	r.Post("/product-links", h.CreateLinks)
	r.Delete("/product-links/{id}", h.DeleteLink)

	r.Get("/orders", h.ListOrders)

	r.Post("/users/{id}/topup", h.TopupUser)
}

func (h *AdminHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.productService.GetAllProducts(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, products)
}

func (h *AdminHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var p model.Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	p.IsActive = true
	if err := h.productService.CreateProduct(r.Context(), &p); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, p)
}

func (h *AdminHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}
	var p model.Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	p.ID = id
	if err := h.productService.UpdateProduct(r.Context(), &p); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "updated"})
}

func (h *AdminHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.productService.DeleteProduct(r.Context(), id); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "deleted"})
}

func (h *AdminHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	productIDStr := r.URL.Query().Get("product_id")
	if productIDStr == "" {
		jsonError(w, "product_id required", http.StatusBadRequest)
		return
	}
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		jsonError(w, "invalid product_id", http.StatusBadRequest)
		return
	}
	_, items, err := h.productService.GetProductWithItems(r.Context(), productID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, items)
}

func (h *AdminHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	var it model.Item
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	it.IsActive = true
	if err := h.productService.CreateItem(r.Context(), &it); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, it)
}

func (h *AdminHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}
	var it model.Item
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	it.ID = id
	if err := h.productService.UpdateItem(r.Context(), &it); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "updated"})
}

func (h *AdminHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.productService.DeleteItem(r.Context(), id); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "deleted"})
}

type createLinksRequest struct {
	ItemID uuid.UUID `json:"item_id"`
	Links  []string  `json:"links"`
}

func (h *AdminHandler) ListLinks(w http.ResponseWriter, r *http.Request) {
	itemIDStr := r.URL.Query().Get("item_id")
	if itemIDStr == "" {
		jsonError(w, "item_id required", http.StatusBadRequest)
		return
	}
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		jsonError(w, "invalid item_id", http.StatusBadRequest)
		return
	}
	links, err := h.productService.GetItemLinks(r.Context(), itemID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	available, total, _ := h.productService.GetLinkStats(r.Context(), itemID)
	jsonResponse(w, map[string]interface{}{
		"links":     links,
		"available": available,
		"total":     total,
	})
}

func (h *AdminHandler) CreateLinks(w http.ResponseWriter, r *http.Request) {
	var req createLinksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Links) == 0 {
		jsonError(w, "links array is empty", http.StatusBadRequest)
		return
	}
	count, err := h.productService.AddLinks(r.Context(), req.ItemID, req.Links)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Tự động thông báo tới toàn bộ người dùng khi admin nạp thêm link vào kho
	go h.notifyUsersStockRestocked(req.ItemID, count)

	jsonResponse(w, map[string]interface{}{
		"status":  "created",
		"created": count,
	})
}

func (h *AdminHandler) DeleteLink(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.productService.DeleteLink(r.Context(), id); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "deleted"})
}

func (h *AdminHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			offset = v
		}
	}
	orders, err := h.orderService.GetAllOrders(r.Context(), limit, offset)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, orders)
}

type topupRequest struct {
	Amount int64  `json:"amount"`
	Note   string `json:"note"`
}

func (h *AdminHandler) TopupUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, "invalid user id", http.StatusBadRequest)
		return
	}
	var req topupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		jsonError(w, "amount must be positive", http.StatusBadRequest)
		return
	}
	if err := h.walletService.AdminTopup(r.Context(), id, req.Amount, req.Note); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "topped up"})
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// notifyUsersStockRestocked broadcasts a restock notification to all users.
func (h *AdminHandler) notifyUsersStockRestocked(itemID uuid.UUID, addedCount int) {
	ctx := context.Background()
	item, availableCount, err := h.productService.GetItemDetail(ctx, itemID)
	if err != nil || item == nil {
		log.Printf("Cannot find item %s for restock notification: %v", itemID, err)
		return
	}

	product, _ := h.productService.GetProduct(ctx, item.ProductID)
	productName := item.Name
	if product != nil {
		productName = fmt.Sprintf("%s — %s", product.Name, item.Name)
	}

	userIDs, err := h.userService.GetAllTelegramIDs(ctx)
	if err != nil {
		log.Printf("Error fetching users for restock notification: %v", err)
		return
	}

	text := fmt.Sprintf("📢 <b>THÔNG BÁO: HÀNG MỚI VỀ KHO!</b>\n"+
		"───────────────────\n"+
		"📦 <b>Sản phẩm:</b> %s\n"+
		"🔢 <b>Vừa nhập thêm:</b> +%d sản phẩm\n"+
		"📊 <b>Tổng tồn kho hiện tại:</b> %d\n"+
		"💵 <b>Đơn giá:</b> %s\n"+
		"───────────────────\n"+
		"⚡ <i>Số lượng có hạn, hãy nhanh tay đặt mua ngay kẻo hết nhé!</i>",
		productName, addedCount, availableCount, bot.FormatMoney(item.Price),
	)

	kb := &bot.InlineKeyboardMarkup{
		InlineKeyboard: [][]bot.InlineKeyboardButton{
			{
				{Text: "🛒 Xem & Mua ngay", CallbackData: fmt.Sprintf("item:%s", item.ID)},
			},
			{
				{Text: "🛍 Danh sách sản phẩm", CallbackData: "products"},
			},
		},
	}

	sentCount := 0
	for _, tgID := range userIDs {
		if err := h.bot.SendMessage(tgID, text, kb); err == nil {
			sentCount++
		}
		// Rate limit: sleep 40ms (~25 messages/sec) to stay well within Telegram limits
		time.Sleep(40 * time.Millisecond)
	}
	log.Printf("Restock notification sent to %d/%d users for item %s (+%d links)", sentCount, len(userIDs), itemID, addedCount)
}
