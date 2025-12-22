package auth

import (
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/JSchatten/go-diploma/internal/storage"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandlers struct {
	storage storage.Storage
}

func NewAuthHandlers(storage storage.Storage) *AuthHandlers {
	return &AuthHandlers{storage: storage}
}

// RegisterHandler регистрирует нового пользователя
func (h *AuthHandlers) RegisterHandler(c *gin.Context) {
	var req struct {
		Login    string `json:"login" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.Error().Err(err).Msg("Invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Хэшируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	userID, err := h.storage.SaveUser(c.Request.Context(), req.Login, string(hashedPassword))
	if err != nil {
		if err == storage.ErrUserExists {
			log.Logger.Warn().Err(err).Msg("User already exists")
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
			return
		}
		log.Logger.Error().Err(err).Msg("Failed to save user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user"})
		return
	}

	token, err := GenerateToken(userID, req.Login)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to generate token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.Header("Authorization", "Bearer "+token)
	c.JSON(http.StatusOK, gin.H{"message": "User registered", "user_id": userID})
}

// LoginHandler аутентифицирует пользователя и выдает токен
func (h *AuthHandlers) LoginHandler(c *gin.Context) {
	var req struct {
		Login    string `json:"login" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.Error().Err(err).Msg("Invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	userID, hash, err := h.storage.GetUserByLogin(c.Request.Context(), req.Login)
	if err != nil {
		if err == storage.ErrUserNotFound {
			log.Logger.Warn().Err(err).Msg("Invalid credentials")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		log.Logger.Error().Err(err).Msg("Failed to authenticate")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to authenticate"})
		return
	}

	// Проверяем пароль
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		log.Logger.Error().Err(err).Msg("Invalid credentials")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := GenerateToken(userID, req.Login)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to generate token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.Header("Authorization", "Bearer "+token)
	c.JSON(http.StatusOK, gin.H{"message": "Login successful", "user_id": userID})
}
