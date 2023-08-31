package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tcmhoang/sservices/business/data/store/user"
	"github.com/tcmhoang/sservices/business/sys/auth"
	"github.com/tcmhoang/sservices/business/sys/database"
	"github.com/tcmhoang/sservices/foundation/keystore"
	"go.uber.org/zap"
)

func genToken(log *zap.SugaredLogger, cfg database.Config, userIDStr string, kid string) error {
	if userIDStr == "" || kid == "" {
		fmt.Println("help: gentoken <user_id> <kid>")
		return ErrHelp
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("passing uuid: %w", err)
	}

	db, err := database.Open(cfg)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store := user.NewStore(log, db)

	usr, err := store.QueryByID(ctx, userID)

	if err != nil {
		return fmt.Errorf("retrieve user: %w", err)
	}

	keysFolder := "zarf/keys/"
	ks, err := keystore.NewFS(os.DirFS(keysFolder))
	if err != nil {
		return fmt.Errorf("reading keys: %w", err)
	}

	// TODO(tcmhoang): move read key info to database
	activeKID := "private"
	a, err := auth.New(activeKID, ks)
	if err != nil {
		return fmt.Errorf("constructing auth: %w", err)
	}
	var roles []auth.Role

	for _, r := range usr.Roles {
		switch strings.ToLower(r) {
		case "admin":
			roles = append(roles, auth.Admin)
		case "user":
			roles = append(roles, auth.User)

		default:

		}
	}

	claims := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "service project",
			Subject:   usr.ID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(8760 * time.Hour).UTC()),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		},
		Roles: roles,
	}

	token, err := a.GenerateToken(claims)
	if err != nil {
		return fmt.Errorf("generating token: %w", err)
	}

	fmt.Printf("-----BEGIN TOKEN-----\n%s\n-----END TOKEN-----\n", token)
	return nil

}
