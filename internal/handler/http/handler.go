package http

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/putteror/thai-id-reader/internal/adapter/pcsc"
	"github.com/putteror/thai-id-reader/internal/service"
)

type WebHandler struct {
	reader    pcsc.Reader
	idService service.IDCardService
}

func NewWebHandler(reader pcsc.Reader, idService service.IDCardService) *WebHandler {
	return &WebHandler{reader: reader, idService: idService}
}

func (h *WebHandler) SetupRoutes(router *gin.Engine) {
	// 0. Use CORS Middleware
	router.Use(CORSMiddleware())

	// 1. Serve the Frontend HTML page
	router.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(indexHTML))
	})

	// 2. API Endpoint to Read Card (State-isolated for each request)
	router.GET("/api/read", func(c *gin.Context) {
		// Attempt to connect to the smart card reader ON-DEMAND
		err := h.reader.Connect()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   true,
				"message": fmt.Sprintf("Reader connection failed: %v", err),
			})
			return
		}
		// ALWAYS disconnect when the HTTP request finishes
		defer h.reader.Disconnect()

		card, err := h.idService.ReadCardData()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   true,
				"message": fmt.Sprintf("Failed to read card data: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"error": false,
			"data":  card,
		})
	})
}

// CORSMiddleware handles Cross-Origin Resource Sharing (CORS)
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Simple embedded HTML for the UI
const indexHTML = `
<!DOCTYPE html>
<html lang="th">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Thai ID Card Reader</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <style>
        body { font-family: 'Sarabun', sans-serif; background-color: #f3f4f6; }
    </style>
</head>
<body class="flex items-center justify-center min-h-screen">

    <div class="bg-white p-8 rounded-xl shadow-lg w-full max-w-lg">
        <div class="flex flex-col items-center">
            
            <h1 class="text-2xl font-bold text-gray-800 mb-6">Thai ID Reader</h1>
            
            <!-- Default Placeholder Image -->
            <div id="photoContainer" class="w-32 h-40 bg-gray-200 rounded-lg overflow-hidden border border-gray-300 shadow-sm flex items-center justify-center mb-6">
                <svg id="photoPlaceholder" class="w-12 h-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"></path>
                </svg>
                <img id="cardPhoto" class="w-full h-full object-cover hidden" src="" />
            </div>

            <!-- Read Button -->
            <button id="readBtn" onclick="readCard()" class="w-full bg-blue-600 hover:bg-blue-700 text-white font-bold py-3 px-4 rounded-lg transition duration-200 flex items-center justify-center">
                <span id="btnText">อ่านข้อมูลบัตร</span>
                <svg id="loadingIcon" class="animate-spin ml-2 h-5 w-5 text-white hidden" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
            </button>

            <!-- Error Message -->
            <div id="errorMsg" class="w-full mt-4 text-center text-red-600 font-medium hidden"></div>

            <!-- Results -->
            <div id="resultContainer" class="w-full mt-6 bg-gray-50 border border-gray-200 rounded-lg p-5 hidden">
                <div class="grid grid-cols-3 gap-2 text-sm">
                    <div class="text-gray-500 font-medium">รหัสประชาชน:</div>
                    <div class="col-span-2 font-bold text-gray-800" id="resID"></div>
                    
                    <div class="text-gray-500 font-medium mt-2">ชื่อ (TH):</div>
                    <div class="col-span-2 font-bold text-gray-800 mt-2" id="resNameTH"></div>

                    <div class="text-gray-500 font-medium mt-2">ชื่อ (EN):</div>
                    <div class="col-span-2 font-bold text-gray-800 mt-2" id="resNameEN"></div>

                    <div class="text-gray-500 font-medium mt-2">วันเกิด:</div>
                    <div class="col-span-2 font-bold text-gray-800 mt-2" id="resDOB"></div>

                    <div class="text-gray-500 font-medium mt-2">เพศ:</div>
                    <div class="col-span-2 font-bold text-gray-800 mt-2" id="resGender"></div>
                </div>
            </div>

        </div>
    </div>

    <script>
        async function readCard() {
            setLoading(true);
            try {
                const response = await fetch('/api/read');
                const result = await response.json();

                if (response.ok && !result.error && result.data) {
                    displayResult(result.data);
                } else {
                    showError(result.message || "Failed to read card.");
                }
            } catch(e) {
                showError("HTTP Connection Error: " + e.message);
            } finally {
                setLoading(false);
            }
        }

        function displayResult(card) {
            document.getElementById('errorMsg').classList.add('hidden');
            document.getElementById('resultContainer').classList.remove('hidden');

            document.getElementById('resID').innerText = card.CitizenID;
            document.getElementById('resNameTH').innerText = card.FullNameTH;
            document.getElementById('resNameEN').innerText = card.FullNameEN;
            
            // Format Date (YYYYMMDD to DD/MM/YYYY string)
            let dob = card.BirthDate;
            if(dob && dob.length === 8) {
                dob = dob.substring(6,8) + "/" + dob.substring(4,6) + "/" + dob.substring(0,4);
            }
            document.getElementById('resDOB').innerText = dob;
            
            // Format Gender
            let gender = card.Gender === "1" ? "ชาย (Male)" : (card.Gender === "2" ? "หญิง (Female)" : card.Gender);
            document.getElementById('resGender').innerText = gender;

            // Display Photo
            if (card.Photo) {
                document.getElementById('photoPlaceholder').classList.add('hidden');
                document.getElementById('cardPhoto').src = "data:image/jpeg;base64," + card.Photo;
                document.getElementById('cardPhoto').classList.remove('hidden');
            }
        }

        function showError(msg) {
            document.getElementById('resultContainer').classList.add('hidden');
            let errEl = document.getElementById('errorMsg');
            errEl.innerText = msg;
            errEl.classList.remove('hidden');
            
            // Reset image
            document.getElementById('photoPlaceholder').classList.remove('hidden');
            document.getElementById('cardPhoto').classList.add('hidden');
            document.getElementById('cardPhoto').src = "";
        }

        function setLoading(isLoading) {
            document.getElementById('readBtn').disabled = isLoading;
            if(isLoading) {
                document.getElementById('btnText').innerText = "กำลังอ่านข้อมูล...";
                document.getElementById('loadingIcon').classList.remove('hidden');
                document.getElementById('readBtn').classList.add('opacity-80', 'cursor-not-allowed');
            } else {
                document.getElementById('btnText').innerText = "อ่านข้อมูลบัตร";
                document.getElementById('loadingIcon').classList.add('hidden');
                document.getElementById('readBtn').classList.remove('opacity-80', 'cursor-not-allowed');
            }
        }
    </script>
</body>
</html>
`
