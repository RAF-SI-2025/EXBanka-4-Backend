package handlers

import (
	"database/sql"

	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/card"
)

type CardServer struct {
	pb.UnimplementedCardServiceServer
	DB        *sql.DB // card_db
	AccountDB *sql.DB // account_db (for cross-DB lookups)
}
