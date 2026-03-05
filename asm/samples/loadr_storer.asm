; loadr_storer.asm - Demonstra LOADR/STORER com endereços além do limite
; de 16 bits do imediato de LOAD/STORE (> 0xFFFF).
;
; Problema: LOAD/STORE usam um imediato de 16 bits, portanto o endereço
; virtual máximo acessível é 0xFFFF (65535). Com segmentação, isso limita
; os acessos de paginação a PD[0]/PT[0..15].
;
; Solução: LOADR/STORER recebem o endereço de um registrador (32 bits),
; permitindo acessar qualquer endereço virtual, incluindo os que exigem
; índices maiores na Page Directory e Page Table.
;
; Este programa:
;   1. Constrói o endereço 0xC042 em W1 via ADD (soma 0x8000 + 0x4042)
;   2. Escreve o valor 0x55 nesse endereço via STORER
;   3. Lê de volta via LOADR e armazena em W2
;
; Resultado esperado: W2 == 0x55

_start:

; === Constrói o endereço alvo 0xC042 em W1 ===
; 0xC042 = 0xC000 + 0x42
; Como o imediato MV é 16 bits, podemos usar 0xC000 diretamente:
MV W1, #0xC000
MV W0, #0x42
ADD W1, W1, W0      ; W1 = 0xC042

; === Escreve 0x55 no endereço W1 ===
MV W0, #0x55
STORER W0, W1       ; memory[W1] = 0x55

; === Lê de volta ===
LOADR W2, W1        ; W2 = memory[W1]

; === W2 deve conter 0x55 ===
HALT
