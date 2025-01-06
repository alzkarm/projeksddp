package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Produk struct {
	ID        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Nama      string `json:"nama"`
	Stok      int    `json:"stok"`
	Terjual   int    `json:"terjual"`
	Gambar    string `json:"gambar"`
	Deskripsi string `json:"deskripsi"`
}

var db *gorm.DB
var produkCache = map[int]Produk{}

func initDatabase() {
	dsn := "root:@tcp(127.0.0.1:3306)/produk_db?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Gagal terhubung ke database")
	}
	fmt.Println("Berhasil terhubung ke database")

	// AutoMigrate untuk memastikan tabel ada
	db.AutoMigrate(&Produk{})
}

func loadProdukToCache() {
	var produkList []Produk
	if err := db.Find(&produkList).Error; err != nil {
		fmt.Println("Gagal memuat data produk:", err)
		return
	}
	for _, produk := range produkList {
		produkCache[produk.ID] = produk
	}
}

func main() {
	initDatabase()
	loadProdukToCache()

	r := gin.Default()

	// GET: Semua Produk
	r.GET("/api/produk", func(c *gin.Context) {
		var produkList []Produk
		if err := db.Find(&produkList).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"pesan": "Gagal mengambil data produk"})
			return
		}
		c.JSON(http.StatusOK, produkList)
	})

	// GET: Produk by ID
	r.GET("/api/produk/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"pesan": "ID harus berupa angka"})
			return
		}

		if produk, exists := produkCache[id]; exists {
			c.JSON(http.StatusOK, produk)
		} else {
			var produk Produk
			if err := db.First(&produk, id).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"pesan": "Produk tidak ditemukan"})
				return
			}
			produkCache[id] = produk
			c.JSON(http.StatusOK, produk)
		}
	})

	// POST: Tambah Produk Baru
	r.POST("/api/produk", func(c *gin.Context) {
		var produkBaru Produk
		if err := c.ShouldBindJSON(&produkBaru); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"pesan": "Input tidak valid"})
			return
		}

		if err := db.Create(&produkBaru).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"pesan": "Gagal menambahkan produk"})
			return
		}

		produkCache[produkBaru.ID] = produkBaru
		c.JSON(http.StatusCreated, produkBaru)
	})

	// DELETE: Hapus Semua Produk dan Reset ID
	r.DELETE("/api/produk", func(c *gin.Context) {
		// Hapus semua data
		if err := db.Exec("DELETE FROM produks").Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"pesan": "Gagal menghapus semua produk"})
			return
		}

		// Reset auto-increment ID ke 1
		if err := db.Exec("ALTER TABLE produks AUTO_INCREMENT = 1").Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"pesan": "Gagal mereset ID produk"})
			return
		}

		// Kosongkan cache
		produkCache = map[int]Produk{}

		c.JSON(http.StatusOK, gin.H{"pesan": "Semua produk berhasil dihapus dan ID telah direset"})
	})

	// PATCH: Update Produk by ID
	r.PATCH("/api/produk/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"pesan": "ID harus berupa angka"})
			return
		}

		var dataUpdate Produk
		if err := c.ShouldBindJSON(&dataUpdate); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"pesan": "Input tidak valid"})
			return
		}

		var produk Produk
		if err := db.First(&produk, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"pesan": "Produk tidak ditemukan"})
			return
		}

		if err := db.Model(&produk).Updates(dataUpdate).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"pesan": "Gagal mengupdate produk"})
			return
		}

		produkCache[produk.ID] = produk
		c.JSON(http.StatusOK, produk)
	})

	// DELETE: Hapus Produk by ID
	r.DELETE("/api/produk/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"pesan": "ID harus berupa angka"})
			return
		}

		// Periksa apakah produk dengan ID tersebut ada di database
		var produk Produk
		if err := db.First(&produk, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"pesan": "Produk tidak ditemukan"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"pesan": "Terjadi kesalahan pada server"})
			}
			return
		}

		// Hapus produk dari database
		if err := db.Delete(&produk).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"pesan": "Gagal menghapus produk"})
			return
		}

		// Hapus dari cache jika ada
		delete(produkCache, id)

		c.JSON(http.StatusOK, gin.H{"pesan": "Produk berhasil dihapus"})
	})

	r.Run(":8000")
}
