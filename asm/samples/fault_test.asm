; fault_test.asm - Força um Page Fault acessando uma página não mapeada
;
; Objetivo: Validar que o mecanismo de Page Fault funciona.
; Monta page tables com PTE[0] mapeada (código) e PTE[4] NÃO mapeada.
; Ao acessar endereço traduzido para PTE[4], o hardware detecta Present=0
; e dispara Page Fault (exception code 3). CR2 recebe o endereço da falha.
;
; Mapeamento de segmentação (antes da paginação):
;   VA 0x1000 → PA 0x4000 (Seg1, armazena Page Directory)
;   VA 0x3000 → PA 0x8000 (Seg3, armazena Page Table)
;
; Mapeamento de paginação:
;   PDE[0]  → PT base 0x8000, Present=1, RW=1
;   PTE[0]  → Frame 0x0000, Present=1 (identity map da página de código)
;   PTE[4]  → 0x00000000, Present=0 (NÃO MAPEADA → Page Fault!)
;
; Fluxo após habilitar paginação:
;   LOAD #0x1123 → Seg1 PA 0x4123 → PD[0] PT[4] → Present=0 → PAGE FAULT
;   Exception handler não é alcançável (endereço fora dos segmentos)
;   → programa encerra após detectar o fault
;
; Saída esperada: "Exception raised: Code 3"

_start:
MV W5, #0x0000

; === Monta PDE[0] = 0x00008003 (little-endian) ===
; PT base = 0x8000, Present=1, RW=1
MV W0, #0x03
STORE W0, #0x1000
MV W0, #0x80
STORE W0, #0x1001
MV W0, #0x00
STORE W0, #0x1002
STORE W0, #0x1003

; === Monta PTE[0] = 0x00000003 (identity map para página de código) ===
; Frame 0x0000, Present=1, RW=1
MV W0, #0x03
STORE W0, #0x3000
MV W0, #0x00
STORE W0, #0x3001
STORE W0, #0x3002
STORE W0, #0x3003

; PTE[4] = 0x00000000 (Present=0, página NÃO mapeada)
; Memória já inicia zerada, mas zeramos explicitamente para clareza
MV W0, #0x00
STORE W0, #0x3010
STORE W0, #0x3011
STORE W0, #0x3012
STORE W0, #0x3013

; === Configura CR3 = 0x4000 (base física da Page Directory) ===
MV CR3, #0x4000

; === Constrói CR0.PG = 0x80000000 (bit 31) ===
; 0x8000 << 16 = 0x80000000 (16 dobros de W0)
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

; Marca W1 com valor sentinela antes do acesso inválido
MV W1, #0xFF

; === Acesso à página NÃO mapeada → PAGE FAULT ===
; VA 0x1123 → Seg1 → PA 0x4123 → PD[0] PT[4] → Present=0 → FAULT!
; CR2 receberá 0x4123 (endereço segmentado que causou a falha)
LOAD W1, #0x1123

; Se a execução chegar aqui, o fault foi tratado e retornou
MV W2, #0x42

HALT
