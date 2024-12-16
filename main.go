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
	ID        int    `json:"id"`
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

func getProdukByID(id int) (Produk, bool) {
	if produk, exists := produkCache[id]; exists {
		return produk, true
	}
	var produk Produk
	if err := db.First(&produk, id).Error; err != nil {
		return Produk{}, false
	}
	produkCache[id] = produk
	return produk, true
}

func main() {
	initDatabase()
	loadProdukToCache()

	r := gin.Default()

	r.GET("/api/produk", func(c *gin.Context) {
		var produkList []Produk
		if err := db.Find(&produkList).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"pesan": "Gagal mengambil data produk"})
			return
		}
		c.JSON(http.StatusOK, produkList)
	})

	r.GET("/api/produk/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"pesan": "ID harus berupa angka"})
			return
		}

		produk, exists := getProdukByID(id)
		if exists {
			c.JSON(http.StatusOK, produk)
		} else {
			c.JSON(http.StatusNotFound, gin.H{"pesan": "Produk tidak ditemukan"})
		}
	})

	r.POST("/api/produk", func(c *gin.Context) {
		var produkBaru Produk
		if err := c.ShouldBindJSON(&produkBaru); err == nil {
			if err := db.Create(&produkBaru).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"pesan": "Gagal menambahkan produk"})
				return
			}
			produkCache[produkBaru.ID] = produkBaru
			c.JSON(http.StatusCreated, produkBaru)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"pesan": "Input tidak valid"})
		}
	})

	r.DELETE("/api/produk/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"pesan": "ID harus berupa angka"})
			return
		}

		if err := db.Delete(&Produk{}, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"pesan": "Produk tidak ditemukan"})
			return
		}

		delete(produkCache, id)

		c.JSON(http.StatusOK, gin.H{"pesan": "Produk berhasil dihapus"})
	})

	r.PATCH("/api/produk/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
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

	r.Run(":8000")
}
