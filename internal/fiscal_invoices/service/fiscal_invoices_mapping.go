package service

import (
	"context"
	"fmt"

	companiesDomain "github.com/ProTrack-Solutions/protrack-api/internal/companies/domain"
	nfeemissionDomain "github.com/ProTrack-Solutions/protrack-api/internal/nfe_emission/domain"
	productsDomain "github.com/ProTrack-Solutions/protrack-api/internal/products/domain"
	saleItemsDomain "github.com/ProTrack-Solutions/protrack-api/internal/sale_items/domain"
	salesDomain "github.com/ProTrack-Solutions/protrack-api/internal/sales/domain"
)

// ATENÇÃO: os códigos de modalidade de determinação da base de cálculo do
// ICMS, CST de PIS/Cofins e mapeamento de forma de pagamento usam valores
// comuns de mercado — validar com um contador antes de emitir em produção.

func (s *Service) buildNFePayload(
	idIntegracao string,
	company companiesDomain.CompanyResponse,
	sale salesDomain.GetSaleByIdRow,
	items []saleItemsDomain.SaleItemRow,
) (nfeemissionDomain.NFePayload, error) {
	notaItens, err := s.buildItens(company, items)
	if err != nil {
		return nfeemissionDomain.NFePayload{}, err
	}

	payload := nfeemissionDomain.NFePayload{
		IDIntegracao:       idIntegracao,
		Presencial:         true,
		ConsumidorFinal:    isConsumidorFinal(sale.BuyerDocument),
		Natureza:           "VENDA",
		Emitente:           nfeemissionDomain.Pessoa{CpfCnpj: onlyDigits(company.Document)},
		Itens:              notaItens,
		Pagamentos:         buildPagamentos(sale),
		ResponsavelTecnico: responsavelTecnico(),
	}

	if sale.BuyerDocument != "" {
		payload.Destinatario = &nfeemissionDomain.Destinatario{
			CpfCnpj: onlyDigits(sale.BuyerDocument),
		}
	}

	return payload, nil
}

func (s *Service) buildNFCePayload(
	idIntegracao string,
	company companiesDomain.CompanyResponse,
	sale salesDomain.GetSaleByIdRow,
	items []saleItemsDomain.SaleItemRow,
) (nfeemissionDomain.NFCePayload, error) {
	notaItens, err := s.buildItens(company, items)
	if err != nil {
		return nfeemissionDomain.NFCePayload{}, err
	}

	return nfeemissionDomain.NFCePayload{
		IDIntegracao:       idIntegracao,
		Natureza:           "VENDA",
		Emitente:           nfeemissionDomain.Pessoa{CpfCnpj: onlyDigits(company.Document)},
		Itens:              notaItens,
		Pagamentos:         buildPagamentos(sale),
		ResponsavelTecnico: responsavelTecnico(),
	}, nil
}

func (s *Service) buildItens(
	company companiesDomain.CompanyResponse,
	items []saleItemsDomain.SaleItemRow,
) ([]nfeemissionDomain.Item, error) {
	notaItens := make([]nfeemissionDomain.Item, 0, len(items))

	for _, item := range items {
		product, err := s.productsService.GetProductById(context.Background(), item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("fiscal_invoices: produto %s não encontrado ao montar item da nota: %w", item.ProductID, err)
		}

		cfop := resolveCFOP(product, company)
		valorTotal := item.UnitPrice * float64(item.Quantity)

		notaItens = append(notaItens, nfeemissionDomain.Item{
			Codigo:    product.ID.String(),
			Descricao: product.Name,
			NCM:       product.Ncm,
			CEST:      product.Cest,
			CFOP:      cfop,
			ValorUnitario: nfeemissionDomain.ValorUnitario{
				Comercial:  item.UnitPrice,
				Tributavel: item.UnitPrice,
			},
			Valor: valorTotal,
			Tributos: nfeemissionDomain.Tributos{
				ICMS: nfeemissionDomain.ICMS{
					Origem: fmt.Sprintf("%d", product.OrigemMercadoria),
					CSOSN:  product.Csosn,
					BaseCalculo: nfeemissionDomain.BaseCalculoICMS{
						ModalidadeDeterminacao: 3, // "valor da operação" — CONFIRMAR com contador
						Valor:                  valorTotal,
					},
					Aliquota: 0,
					Valor:    0,
				},
				PIS: nfeemissionDomain.PIS{
					CST:         "99",
					BaseCalculo: 0,
					Aliquota:    0,
					Valor:       0,
				},
				Cofins: nfeemissionDomain.Cofins{
					CST:         "99",
					BaseCalculo: 0,
					Aliquota:    0,
					Valor:       0,
				},
			},
		})
	}

	return notaItens, nil
}

// buildCadastrarEmpresaPayload monta o payload de POST /empresa a partir dos
// dados já cadastrados da empresa no ProTrack. CodigoCidade (código IBGE do
// município) fica vazio — a tabela companies não guarda esse dado hoje, só o
// nome da cidade. Sem ele, o PlugNotas provavelmente rejeita o cadastro;
// adicionar um campo address_city_code (ou resolver via CEP/IBGE) é um
// próximo passo necessário para este fluxo funcionar de ponta a ponta.
func (s *Service) buildCadastrarEmpresaPayload(company companiesDomain.CompanyResponse, providerCertID string) (nfeemissionDomain.CadastrarEmpresaPayload, error) {
	if company.RegimeTributario == nil {
		return nfeemissionDomain.CadastrarEmpresaPayload{}, fmt.Errorf("fiscal_invoices: regime_tributario não preenchido para a empresa")
	}
	simplesNacional := fmt.Sprintf("%v", company.RegimeTributario) == "simples_nacional"

	inscricaoEstadual := company.InscricaoEstadual
	if company.InscricaoEstadualIsento {
		inscricaoEstadual = "ISENTO"
	} else if inscricaoEstadual == "" {
		return nfeemissionDomain.CadastrarEmpresaPayload{}, fmt.Errorf("fiscal_invoices: inscrição estadual não preenchida para a empresa")
	}

	return nfeemissionDomain.CadastrarEmpresaPayload{
		CpfCnpj:           onlyDigits(company.Document),
		InscricaoEstadual: inscricaoEstadual,
		RazaoSocial:       company.Name,
		NomeFantasia:      company.TradeName,
		Certificado:       providerCertID,
		SimplesNacional:   simplesNacional,
		RegimeTributario:  1, // 1 = Simples Nacional (único regime suportado hoje — ver fiscal_regime_enum)
		Endereco: nfeemissionDomain.EnderecoEmpresa{
			CodigoPais:      "1058",
			DescricaoPais:   "Brasil",
			Logradouro:      company.AddressStreet,
			Numero:          company.AddressNumber,
			Complemento:     company.AddressComplement,
			Bairro:          company.AddressNeighborhood,
			CodigoCidade:    "", // TODO: código IBGE do município — ver comentário acima
			DescricaoCidade: company.AddressCity,
			Estado:          company.AddressState,
			CEP:             onlyDigits(company.AddressZipcode),
		},
		NFe:  moduloConfig(s.cfg.IsProduction),
		NFCe: moduloConfig(s.cfg.IsProduction),
	}, nil
}

// moduloConfig ativa o módulo (NF-e/NFC-e) no PlugNotas, usando o ambiente de
// produção da SEFAZ somente quando IS_PRODUCTION=true — em qualquer outro
// caso, fica em homologação, para não emitir nota fiscal real por engano.
func moduloConfig(isProduction bool) *nfeemissionDomain.ModuloConfig {
	cfg := &nfeemissionDomain.ModuloConfig{Ativo: true}
	cfg.Config.Producao = isProduction
	return cfg
}

// isConsumidorFinal decide se a venda é para consumidor final. Documento com
// 14 dígitos (CNPJ) indica compra por pessoa jurídica — tratada aqui como
// possível revenda, então NÃO é consumidor final. Documento com 11 dígitos
// (CPF) ou venda sem documento (balcão) é tratada como consumidor final, o
// caso comum de varejo. Não é uma regra perfeita (uma PJ também pode ser
// consumidora final, ex.: compra para uso próprio) — o ideal seria coletar
// essa informação explicitamente na venda; até lá, este é o padrão mais
// seguro para o cenário de varejo do ProTrack.
func isConsumidorFinal(buyerDocument string) bool {
	return len(onlyDigits(buyerDocument)) != 14
}

// resolveCFOP escolhe o CFOP de saída do produto. TODO: comparar o estado do
// destinatário (quando houver) com company.AddressState para decidir
// dentro/fora do estado de fato, em vez de preferir sempre "dentro".
func resolveCFOP(product productsDomain.ProductResponse, company companiesDomain.CompanyResponse) string {
	if product.CfopSaidaDentroEstado != "" {
		return product.CfopSaidaDentroEstado
	}
	if product.CfopSaidaForaEstado != "" {
		return product.CfopSaidaForaEstado
	}
	return "5102"
}

// buildPagamentos mapeia a forma de pagamento da venda para os códigos de
// "meio" da tabela SEFAZ. CONFIRMAR lista completa com a doc do PlugNotas.
func buildPagamentos(sale salesDomain.GetSaleByIdRow) []nfeemissionDomain.Pagamento {
	meio := "99"
	switch fmt.Sprintf("%v", sale.PaymentMethod) {
	case "cash":
		meio = "01"
	case "credit_card":
		meio = "03"
	case "debit_card":
		meio = "04"
	case "pix":
		meio = "17"
	}

	return []nfeemissionDomain.Pagamento{
		{
			AVista: fmt.Sprintf("%v", sale.PaymentMethod) != "installments",
			Meio:   meio,
			Valor:  sale.TotalAmount,
		},
	}
}

// responsavelTecnico identifica a ProTrack como responsável técnico pela
// emissão (grupo infRespTec, exigido pela SEFAZ). TODO: mover para o config
// em vez de hardcoded, e preencher com o CNPJ/e-mail reais da empresa.
func responsavelTecnico() nfeemissionDomain.ResponsavelTecnico {
	return nfeemissionDomain.ResponsavelTecnico{
		CpfCnpj: "00000000000000",
		Nome:    "ProTrack Solutions",
		Email:   "contato@protrack.com.br",
		Telefone: nfeemissionDomain.Telefone{
			DDD:    "00",
			Numero: "000000000",
		},
	}
}
