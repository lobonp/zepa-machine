; Sample: habilita paginação montando PDE/PTE manualmente em memória
; Contexto de segmentação (MMU padrão):
; - VA 0x1000 -> PA 0x4000 (escrevemos a Page Directory)
; - VA 0x3000 -> PA 0x8000 (escrevemos a Page Table)
; - VA 0x2xxx -> PA 0xBxxx (frame físico mapeado)

MV W5, #0x0000

; PDE[0] = 0x00008003 (PT base = 0x8000, Present=1, RW=1)
MV W0, #0x03
STORE W0, #0x1000
MV W0, #0x80
STORE W0, #0x1001
MV W0, #0x00
STORE W0, #0x1002
STORE W0, #0x1003

; PTE[0] = 0x00000003 (mapeamento identidade para a página de código)
MV W0, #0x03
STORE W0, #0x3000
MV W0, #0x00
STORE W0, #0x3001
MV W0, #0x00
STORE W0, #0x3002
STORE W0, #0x3003

; PTE[4] = 0x0000B003 (segmented VA 0x4xxx -> frame físico 0xB000)
MV W0, #0x03
STORE W0, #0x3010
MV W0, #0xB0
STORE W0, #0x3011
MV W0, #0x00
STORE W0, #0x3012
STORE W0, #0x3013

; Pré-carrega um byte no frame físico 0xB123 (via VA 0x2123)
MV W0, #0x2A
STORE W0, #0x2123

; CR3 = 0x4000 (base física da Page Directory)
MV CR3, #0x4000

; Monta CR0.PG (bit 31) em W0: 0x8000 << 16 = 0x80000000
MV W0, #0x8000
ADD W0, W0, W0
ADD W0, W0, W0
ADD W0, W0, W0
ADD W0, W0, W0
ADD W0, W0, W0
ADD W0, W0, W0
ADD W0, W0, W0
ADD W0, W0, W0
ADD W0, W0, W0
ADD W0, W0, W0
ADD W0, W0, W0
ADD W0, W0, W0
ADD W0, W0, W0
ADD W0, W0, W0
ADD W0, W0, W0
ADD W0, W0, W0

; CR0 = CR0 | 0x80000000  (CR0 inicia em 0)
ADD CR0, W0, W5

; A partir daqui, LOAD/STORE passam pelo page walk
; VA 0x1123 -> segmented 0x4123 -> PTE[4] -> PA 0xB123
LOAD W1, #0x1123

; Escrita/leitura virtual na mesma página mapeada (page 0)
MV W2, #0x7B
STORE W2, #0x1456
LOAD W3, #0x1456

HALT