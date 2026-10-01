package demo

// Dados fixos da empresa demo: "Casa & Obra", uma loja de materiais de
// construção. Os valores foram escolhidos para que os gráficos do dashboard e
// do fluxo de caixa tenham margem positiva e algumas contas em atraso.

type productCategorySeed struct {
	Name  string
	Color string
}

var productCategories = []productCategorySeed{
	{"Cimento e Argamassa", "#94A3B8"},
	{"Hidráulica", "#3B82F6"},
	{"Elétrica", "#F59E0B"},
	{"Tintas", "#EC4899"},
	{"Ferramentas", "#EF4444"},
	{"Ferragens", "#10B981"},
}

type productSeed struct {
	Name       string
	Category   string
	Unit       string // unit_of_measure: UN, KG, G, L, ML
	SellInBulk bool
	Cost       float64
	Price      float64
	Quantity   int // estoque atual (ignorado quando SellInBulk)
	Weight     int // popularidade relativa nas vendas
	MaxQty     int // quantidade máxima por item de venda
}

var products = []productSeed{
	{"Cimento CP II 50kg", "Cimento e Argamassa", "UN", false, 29.90, 39.90, 180, 30, 10},
	{"Argamassa AC-I 20kg", "Cimento e Argamassa", "UN", false, 12.50, 18.90, 140, 18, 8},
	{"Argamassa AC-III 20kg", "Cimento e Argamassa", "UN", false, 26.00, 37.90, 60, 8, 6},
	{"Cal Hidratada 20kg", "Cimento e Argamassa", "UN", false, 14.00, 21.50, 75, 8, 6},
	{"Rejunte Flexível 1kg", "Cimento e Argamassa", "UN", false, 7.20, 12.90, 2, 10, 5},

	{"Tubo PVC Soldável 25mm 6m", "Hidráulica", "UN", false, 18.50, 28.90, 90, 12, 6},
	{"Tubo PVC Esgoto 100mm 6m", "Hidráulica", "UN", false, 52.00, 79.90, 40, 6, 3},
	{"Joelho 90° PVC 25mm", "Hidráulica", "UN", false, 0.90, 2.20, 400, 14, 12},
	{"Registro de Gaveta 3/4\"", "Hidráulica", "UN", false, 34.00, 54.90, 25, 5, 2},
	{"Caixa d'Água 1000L", "Hidráulica", "UN", false, 380.00, 549.90, 6, 2, 1},
	{"Torneira de Cozinha Bica Móvel", "Hidráulica", "UN", false, 42.00, 69.90, 18, 4, 2},

	{"Cabo Flexível 2,5mm 100m", "Elétrica", "UN", false, 189.00, 279.90, 30, 5, 2},
	{"Disjuntor Bipolar 32A", "Elétrica", "UN", false, 28.00, 45.90, 1, 5, 2},
	{"Tomada 2P+T 10A", "Elétrica", "UN", false, 6.50, 12.90, 220, 14, 10},
	{"Lâmpada LED 12W", "Elétrica", "UN", false, 5.80, 11.90, 300, 16, 10},
	{"Quadro de Distribuição 12 Disjuntores", "Elétrica", "UN", false, 62.00, 98.90, 12, 3, 1},

	{"Tinta Acrílica Fosca Branco 18L", "Tintas", "UN", false, 189.00, 289.90, 35, 9, 2},
	{"Tinta Acrílica Fosca Branco 3,6L", "Tintas", "UN", false, 52.00, 84.90, 48, 9, 3},
	{"Massa Corrida 25kg", "Tintas", "UN", false, 48.00, 74.90, 30, 6, 3},
	{"Selador Acrílico 18L", "Tintas", "UN", false, 98.00, 149.90, 3, 4, 2},
	{"Rolo de Lã 23cm", "Tintas", "UN", false, 14.00, 24.90, 65, 8, 3},

	{"Furadeira de Impacto 650W", "Ferramentas", "UN", false, 189.00, 299.90, 9, 2, 1},
	{"Trena 5m", "Ferramentas", "UN", false, 9.50, 19.90, 70, 6, 2},
	{"Nível de Alumínio 40cm", "Ferramentas", "UN", false, 18.00, 32.90, 22, 4, 1},
	{"Colher de Pedreiro 8\"", "Ferramentas", "UN", false, 11.00, 19.90, 40, 5, 2},
	{"Carrinho de Mão 60L", "Ferramentas", "UN", false, 165.00, 249.90, 8, 2, 1},

	{"Prego 17x27", "Ferragens", "KG", true, 14.00, 22.90, 0, 8, 5},
	{"Arame Recozido 18", "Ferragens", "KG", true, 16.00, 26.90, 0, 5, 4},
	{"Parafuso com Bucha 8mm (cento)", "Ferragens", "UN", false, 18.00, 32.90, 55, 6, 3},
	{"Dobradiça 3\" (par)", "Ferragens", "UN", false, 7.50, 14.90, 4, 5, 4},
}

type vendorSeed struct {
	Name     string
	Email    string
	Phone    string
	City     string
	Category string // categoria de produto que o fornecedor abastece
}

var vendors = []vendorSeed{
	{"Distribuidora Cimentos Paulista Ltda", "vendas@cimentospaulista.example.com", "1932410001", "Sumaré", "Cimento e Argamassa"},
	{"HidroSul Materiais Hidráulicos Ltda", "comercial@hidrosul.example.com", "1932410002", "Campinas", "Hidráulica"},
	{"EletroCamp Distribuidora Ltda", "pedidos@eletrocamp.example.com", "1932410003", "Campinas", "Elétrica"},
	{"Cores & Cia Tintas Ltda", "atendimento@coresecia.example.com", "1932410004", "Valinhos", "Tintas"},
	{"Ferramentas Alvorada Comércio Ltda", "vendas@alvorada.example.com", "1932410005", "Hortolândia", "Ferramentas"},
}

var billCategories = []struct {
	Name        string
	Description string
}{
	{"Fornecedores", "Compra de mercadorias para revenda"},
	{"Aluguel", "Aluguel do galpão da loja"},
	{"Salários", "Folha de pagamento dos funcionários"},
	{"Energia", "Conta de energia elétrica"},
	{"Água", "Conta de água e esgoto"},
	{"Internet e Telefone", "Plano de internet e telefonia"},
	{"Impostos", "Simples Nacional (DAS)"},
	{"Marketing", "Anúncios e panfletagem"},
}

// recurringBill é uma conta que se repete todo mês.
type recurringBill struct {
	Category    string
	Description string
	Day         int
	Amount      float64
	Variation   float64 // variação percentual máxima (+/-) sobre Amount
}

var recurringBills = []recurringBill{
	{"Aluguel", "Aluguel do galpão", 5, 4500.00, 0},
	{"Salários", "Folha de pagamento", 5, 12400.00, 0.03},
	{"Energia", "Conta de energia", 12, 640.00, 0.15},
	{"Água", "Conta de água", 15, 185.00, 0.15},
	{"Internet e Telefone", "Internet fibra + telefone", 20, 199.90, 0},
	{"Impostos", "DAS - Simples Nacional", 20, 2300.00, 0.12},
}

var customerNames = []struct {
	Name   string
	Gender string // gender_enum
}{
	{"Ana Paula Ribeiro", "FEMALE"},
	{"Bruno Henrique Costa", "MALE"},
	{"Carla Mendes Oliveira", "FEMALE"},
	{"Daniel Souza Martins", "MALE"},
	{"Eduarda Lima Ferreira", "FEMALE"},
	{"Fábio Augusto Pereira", "MALE"},
	{"Gabriela Santos Rocha", "FEMALE"},
	{"Henrique Alves Barbosa", "MALE"},
	{"Isabela Cardoso Nunes", "FEMALE"},
	{"João Victor Teixeira", "MALE"},
	{"Juliana Moreira Dias", "FEMALE"},
	{"Leonardo Araújo Gomes", "MALE"},
	{"Mariana Castro Lopes", "FEMALE"},
	{"Marcelo Vieira Campos", "MALE"},
	{"Natália Freitas Ramos", "FEMALE"},
	{"Otávio Correia Pinto", "MALE"},
	{"Patrícia Monteiro Reis", "FEMALE"},
	{"Rafael Nascimento Cruz", "MALE"},
	{"Renata Azevedo Melo", "FEMALE"},
	{"Rodrigo Batista Farias", "MALE"},
	{"Sabrina Duarte Cunha", "FEMALE"},
	{"Thiago Rezende Moura", "MALE"},
	{"Vanessa Peixoto Brito", "FEMALE"},
	{"Wagner Fonseca Lima", "MALE"},
	{"Construtora Bom Lar (Paulo Sérgio)", "NOT_SAY"},
}

var neighborhoods = []string{
	"Cambuí", "Taquaral", "Barão Geraldo", "Jardim Proença", "Castelo",
	"Botafogo", "Nova Campinas", "Jardim Chapadão", "Vila Industrial", "Guanabara",
}

var streets = []string{
	"Rua das Acácias", "Avenida Brasil", "Rua Barão de Jaguara", "Rua José Paulino",
	"Avenida Orosimbo Maia", "Rua Coronel Quirino", "Rua Maria Monteiro",
	"Avenida Norte-Sul", "Rua Dr. Quirino", "Rua Conceição",
}
