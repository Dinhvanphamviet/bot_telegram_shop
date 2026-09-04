package service

import (
	"context"

	"telegram-shop/internal/model"
	"telegram-shop/internal/repository"

	"github.com/google/uuid"
)

// ProductService handles product/item business logic.
type ProductService struct {
	productRepo     *repository.ProductRepo
	itemRepo        *repository.ItemRepo
	productLinkRepo *repository.ProductLinkRepo
}

// NewProductService creates a new ProductService.
func NewProductService(productRepo *repository.ProductRepo, itemRepo *repository.ItemRepo, productLinkRepo *repository.ProductLinkRepo) *ProductService {
	return &ProductService{
		productRepo:     productRepo,
		itemRepo:        itemRepo,
		productLinkRepo: productLinkRepo,
	}
}

// ListProducts returns all active products.
func (s *ProductService) ListProducts(ctx context.Context) ([]model.Product, error) {
	return s.productRepo.FindAllActive(ctx)
}

// GetProductWithItems returns a product and its items with stock info.
func (s *ProductService) GetProductWithItems(ctx context.Context, productID uuid.UUID) (*model.Product, []model.ItemWithStock, error) {
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, nil, err
	}
	if product == nil {
		return nil, nil, nil
	}

	items, err := s.itemRepo.FindByProductID(ctx, productID)
	if err != nil {
		return nil, nil, err
	}

	return product, items, nil
}

// GetItemDetail returns an item with its available link count.
func (s *ProductService) GetItemDetail(ctx context.Context, itemID uuid.UUID) (*model.Item, int, error) {
	item, err := s.itemRepo.FindByID(ctx, itemID)
	if err != nil {
		return nil, 0, err
	}
	if item == nil {
		return nil, 0, nil
	}

	count, err := s.itemRepo.CountAvailableLinks(ctx, itemID)
	if err != nil {
		return nil, 0, err
	}

	return item, count, nil
}

// --- Admin operations ---

// CreateProduct creates a new product (admin).
func (s *ProductService) CreateProduct(ctx context.Context, p *model.Product) error {
	return s.productRepo.Create(ctx, p)
}

// UpdateProduct updates a product (admin).
func (s *ProductService) UpdateProduct(ctx context.Context, p *model.Product) error {
	return s.productRepo.Update(ctx, p)
}

// DeleteProduct deletes a product (admin).
func (s *ProductService) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	return s.productRepo.Delete(ctx, id)
}

// CreateItem creates a new item (admin).
func (s *ProductService) CreateItem(ctx context.Context, it *model.Item) error {
	return s.itemRepo.Create(ctx, it)
}

// UpdateItem updates an item (admin).
func (s *ProductService) UpdateItem(ctx context.Context, it *model.Item) error {
	return s.itemRepo.Update(ctx, it)
}

// DeleteItem deletes an item (admin).
func (s *ProductService) DeleteItem(ctx context.Context, id uuid.UUID) error {
	return s.itemRepo.Delete(ctx, id)
}

// AddLinks adds multiple links to an item (admin).
func (s *ProductService) AddLinks(ctx context.Context, itemID uuid.UUID, links []string) (int, error) {
	return s.productLinkRepo.CreateBatch(ctx, itemID, links)
}

// GetItemLinks returns all links for an item (admin).
func (s *ProductService) GetItemLinks(ctx context.Context, itemID uuid.UUID) ([]model.ProductLink, error) {
	return s.productLinkRepo.FindByItemID(ctx, itemID)
}

// GetAllProducts returns all products including inactive (admin).
func (s *ProductService) GetAllProducts(ctx context.Context) ([]model.Product, error) {
	return s.productRepo.FindAll(ctx)
}

// GetProduct returns a single product by ID.
func (s *ProductService) GetProduct(ctx context.Context, id uuid.UUID) (*model.Product, error) {
	return s.productRepo.FindByID(ctx, id)
}

// GetItem returns a single item by ID.
func (s *ProductService) GetItem(ctx context.Context, id uuid.UUID) (*model.Item, error) {
	return s.itemRepo.FindByID(ctx, id)
}

// DeleteLink removes a single link (admin).
func (s *ProductService) DeleteLink(ctx context.Context, id uuid.UUID) error {
	return s.productLinkRepo.Delete(ctx, id)
}

// GetLinkStats returns available and total link counts for an item.
func (s *ProductService) GetLinkStats(ctx context.Context, itemID uuid.UUID) (available, total int, err error) {
	return s.productLinkRepo.CountByItemID(ctx, itemID)
}
