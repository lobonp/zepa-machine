; paging_two_frames.asm - Mapeia duas páginas virtuais para frames físicos distintos
;
; Demonstra que o mecanismo de paginação permite isolar dados em frames
; diferentes, acessíveis por endereços virtuais distintos.
;
; Mapeamento de segmentação:
;   Seg0: VA 0x0xxx → PA 0x0xxx  (código, identity)
;   Seg1: VA 0x1xxx → PA 0x4xxx  (dados do usuário)
;   Seg2: VA 0x2xxx → PA 0xBxxx  (stack, grows negative)
;   Seg3: VA 0x3xxx → PA 0x8xxx  (dados do kernel / page tables)
;
; Mapeamento de paginação:
;   PDE[0]  → PT base 0x8000
;   PTE[0]  → Frame 0x0000 (identity, código)
;   PTE[4]  → Frame 0xA000 (dados via Seg1: VA 0x1xxx → PA 0x4xxx → PT[4])
;   PTE[11] → Frame 0xD000 (dados via Seg2: VA 0x2xxx → PA 0xBxxx → PT[11])
;
; Teste:
;   Escreve 0x41 ('A') no frame 0xA000 via VA 0x1000
;   Escreve 0x44 ('D') no frame 0xD000 via VA 0x2000
;   Lê de volta ambos e compara → dados isolados em frames diferentes

_start:
MV W5, #0x0000

; === PDE[0] = 0x00008003 (PT base = 0x8000, P=1, RW=1) ===
MV W0, #0x03
STORE W0, #0x1000
MV W0, #0x80
STORE W0, #0x1001
MV W0, #0x00
STORE W0, #0x1002
STORE W0, #0x1003

; === PTE[0] = 0x00000003 (identity map, página de código) ===
MV W0, #0x03
STORE W0, #0x3000
MV W0, #0x00
STORE W0, #0x3001
STORE W0, #0x3002
STORE W0, #0x3003

; === PTE[4] = 0x0000A003 (frame 0xA000, P=1, RW=1) ===
; Endereço na PT: base 0x8000 + 4*4 = 0x8010 → VA 0x3010 (Seg3)
MV W0, #0x03
STORE W0, #0x3010
MV W0, #0xA0
STORE W0, #0x3011
MV W0, #0x00
STORE W0, #0x3012
STORE W0, #0x3013

; === PTE[11] = 0x0000D003 (frame 0xD000, P=1, RW=1) ===
; Endereço na PT: base 0x8000 + 11*4 = 0x802C → VA 0x302C (Seg3)
MV W0, #0x03
STORE W0, #0x302C
MV W0, #0xD0
STORE W0, #0x302D
MV W0, #0x00
STORE W0, #0x302E
STORE W0, #0x302F

; === CR3 = 0x4000 (base física da Page Directory) ===
MV CR3, #0x4000

; === Constrói CR0.PG = 0x80000000 ===
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

; === Habilita paginação ===
ADD CR0, W0, W5

; === Escreve no Frame A (0xA000) via VA 0x1000 ===
; VA 0x1000 → Seg1 PA 0x4000 → PD[0] PT[4] → frame 0xA000
MV W0, #0x41
STORE W0, #0x1000

; === Escreve no Frame D (0xD000) via VA 0x2000 ===
; VA 0x2000 → Seg2 PA 0xB000 → PD[0] PT[11] → frame 0xD000
MV W0, #0x44
STORE W0, #0x2000

; === Lê de volta dos dois frames ===
LOAD W1, #0x1000
LOAD W2, #0x2000

; W1 deve conter 0x41 (65), W2 deve conter 0x44 (68)
; Dados em frames físicos diferentes, acessados por VAs diferentes
CMP W1, W2

HALT
