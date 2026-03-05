MV W0, 0      ; 0
MV W1, #0x1000 ; 4 - destVA (Seg1)
MV W5, 0      ; 8 - zero para comparação
MV W4, 0      ; 12

D2M W0, 0     ; 16
CMP W4, W5    ; 20 - compara bytesWritten com 0
BGT #0x1000   ; 24 - se W4 > 0 (sucesso), pula para programa
HALT          ; 28 - se falhou



JUMP 32       ;36