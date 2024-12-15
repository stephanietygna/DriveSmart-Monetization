import csv
import re

# Defina o caminho para o arquivo de log e o arquivo CSV de saída
log_file_path = 'containerlog.txt'
csv_file_path = 'log.csv'

# Função para extrair dados do log
def extract_data(line):
    match = re.match(r'(\d{4}/\d{2}/\d{2}) (\d{2}:\d{2}:\d{2}) (.*): (.*)', line)
    if match:
        return match.groups()
    return None

# Inicialize a lista para armazenar os dados do CSV
csv_data = []

# Abra o arquivo de log para leitura
with open(log_file_path, 'r') as log_file:
    # Leia todas as linhas do arquivo de log
    log_lines = log_file.readlines()

# Dicionário para armazenar os dados temporariamente
current_data = {
    'Data': '',
    'Hora': '',
    'Curva Brusca': '',
    'Créditos': '',
    'Zigue Zague': '',
    'Aceleração Anômala': ''
}

# Contadores para zigue zague e aceleração anômala
zigue_zague_counter = 0
anomalia_counter = 0

# Itere sobre as linhas do log e extraia os dados
for line in log_lines:
    data = extract_data(line)
    if data:
        date, time, key, value = data
        current_data['Data'] = date
        current_data['Hora'] = time
        if key == 'Curva brusca':
            if value == 'Direção neutra':
                current_data['Curva Brusca'] = 'false'
            else:
                current_data['Curva Brusca'] = value
        elif key == 'Créditos':
            current_data['Créditos'] = value
        elif key == 'Zigue-zague':
            current_data['Zigue Zague'] = value
            zigue_zague_counter = 0  # Reset counter
        elif key == 'Anomalia':
            current_data['Aceleração Anômala'] = value
            anomalia_counter = 0  # Reset counter

        # Adicione os dados completos à lista csv_data quando todos os campos obrigatórios estiverem preenchidos
        if current_data['Data'] and current_data['Hora'] and current_data['Curva Brusca'] and current_data['Créditos']:
            # Preencha com strings vazias se não houver dados para zigue zague e aceleração anômala a cada 10 linhas
            if zigue_zague_counter >= 10:
                current_data['Zigue Zague'] = ''
                zigue_zague_counter = 0
            if anomalia_counter >= 10:
                current_data['Aceleração Anômala'] = ''
                anomalia_counter = 0

            csv_data.append(current_data.copy())
            # Reinicie os dados temporários, exceto os contadores
            current_data = {
                'Data': '',
                'Hora': '',
                'Curva Brusca': '',
                'Créditos': '',
                'Zigue Zague': '',
                'Aceleração Anômala': ''
            }
            zigue_zague_counter += 1
            anomalia_counter += 1

# Escreva os dados no arquivo CSV
with open(csv_file_path, 'w', newline='') as csv_file:
    writer = csv.DictWriter(csv_file, fieldnames=current_data.keys())
    # Escreva o cabeçalho do CSV
    writer.writeheader()
    # Escreva os dados extraídos
    writer.writerows(csv_data)
