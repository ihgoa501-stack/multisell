# Competitor Sources - 2026-06

Retrieval date: 2026-06-15

## Evidence Rules

Evidence tags:

- `official_site`: vendor public website.
- `official_help`: vendor public help center or docs.
- `official_pricing`: vendor public pricing page.
- `case_study`: vendor-published customer story.
- `inference`: conclusion derived from sourced claims.

Roadmap decisions should not depend on claims that cannot be traced to a source or a clearly marked inference.

## Source Table

| Competitor | URL | Evidence Type | Useful Claims For LingMirror |
| --- | --- | --- | --- |
| Mabang | https://www.mabangerp.com/ | official_site | Full-process ERP, ERP/WMS/TMS/SCM product line, logistics ecosystem, TMS reconciliation, WMS/PDA, Amazon profit/reporting/FBA replenishment |
| Dianxiaomi | https://www.dianxiaomi.com/ | official_site | Lightweight multi-platform ERP, 70+ platforms, 1700+ logistics, order processing, warehouse, logistics tracking, profit reports, permissions, operation logs |
| Eccang | https://www.eccang.com/ | official_site | Multi-platform order management, overseas warehouse, finance automation, gross/net profit, first-leg/FBA shipment, listing lifecycle monitoring |
| Tongtool | https://www.tongtool.com/ | official_site | Full-process ERP, listing, WMS, logistics platform, PaaS/microservice architecture, open API, 1000+ logistics/overseas warehouse providers |
| SellFox | https://www.sellfox.com/ | official_site | Amazon/multi-platform refined operations, 200+ platform fee details, FIFO order cost, funds tracking, platform-specific profit reports, logistics comparison, overseas warehouse/FBA inventory sync |
| Lingxing | https://www.lingxing.com/ | official_site | Amazon ERP, multi-platform ERP, overseas warehouse WMS, 40+ mainstream platforms, finance control positioning |
| CaptainBI | https://www.captainbi.com/ | official_site | Amazon BI, finance/VAT, advertising, FBA inventory, AI diagnosis, AI advertising, multi-dimensional reports, intelligent replenishment |
| BigSeller | https://www.bigseller.com/ | official_site | Free omnichannel ecommerce solution, 20+ Southeast Asia platforms, product/order/inventory/purchase/marketing/report modules, automation |
| 4Seller | https://www.4seller.com/ | official_site | Listing, order, inventory, Amazon MCF, shipping carriers, Amazon profit calculator, bulk edit |
| Sellercloud | https://sellercloud.com/ | official_site | Catalog, inventory, warehouse, order rule engine, purchasing, shipping, reporting, accounting, web service API, 350+ integrations |
| ShipStation | https://www.shipstation.com/ | official_site | Carrier integrations, marketplace integrations, shipping labels, rate comparison, address validation, tracking, analytics APIs |
| Linnworks | https://www.linnworks.com/ | official_site | Order/shipping management, inventory, warehouse, marketplace listings, analytics, automation, carrier labels/tracking/cost control, bulk listing |

## Claim Log

| Competitor | Claim | Evidence Type | Source | LingMirror Product Implication |
| --- | --- | --- | --- | --- |
| Mabang | Presents product development, order, supply chain, product management, inventory planning, logistics, overseas warehouse, finance reports, customer service, and refined operations as a full-process solution. | official_site | https://www.mabangerp.com/ | Benchmark the business graph, not the screen layout. |
| Mabang | TMS includes order tracking, in-transit feedback, bill generation, automatic reconciliation, and multiple logistics price systems. | official_site | https://www.mabangerp.com/ | Build freight bill reconciliation after shipping snapshots. |
| Mabang | WMS highlights bin setup, barcode operation, PDA, paperless picking, inventory checks, and multiple outbound modes. | official_site | https://www.mabangerp.com/ | Delay full WMS/PDA until warehouse and inventory fundamentals are reliable. |
| Mabang | Amazon edition highlights multi-dimensional profit reports, ad automation, and FBA replenishment suggestions. | official_site | https://www.mabangerp.com/ | Amazon-specific depth belongs behind a real Amazon adapter. |
| Dianxiaomi | Claims 70+ platforms, 1700+ logistics for automatic tracking-number retrieval, 130+ overseas warehouses, order processing, profit reports, purchase management, warehouse management, logistics tracking, permissions, and operation logs. | official_site | https://www.dianxiaomi.com/ | Copy low-friction workflows and audit basics; do not copy shallow cost depth as the final finance model. |
| Eccang | Claims multi-platform order handling, shipping-channel matching, finance automation, product/order/staff/store/platform gross profit, fee import for net profit, first-leg/FBA shipment, and listing lifecycle monitoring. | official_site | https://www.eccang.com/ | Build net-profit foundations through settlement import, freight actuals, and allocated first-leg cost. |
| Tongtool | Claims ERP, Listing, WMS, logistics platform, supply-chain distribution, open data interfaces, microservice/PaaS architecture, PDA warehouse operations, and 1000+ logistics/overseas warehouse providers. | official_site | https://www.tongtool.com/ | Preserve module boundaries and adapter-first design. |
| SellFox | Claims 200+ platform fee details, FIFO order-cost accounting, funds tracking from order to remittance, platform-specific profit reports, logistics comparison, 17TRACK integration, and inventory sync across local/overseas/FBA warehouses. | official_site | https://www.sellfox.com/ | Build finance around source-specific cost layers and settlement lineage. |
| Lingxing | Positions separate Amazon ERP, multi-platform ERP, and overseas warehouse WMS products while claiming 40+ mainstream cross-border platforms. | official_site | https://www.lingxing.com/ | Keep marketplace-specific capabilities separate from generic ERP core. |
| CaptainBI | Presents Amazon sales management, finance/VAT, advertising, FBA inventory, AI diagnosis, AI advertising, intelligent replenishment, and 80+ reports. | official_site | https://www.captainbi.com/ | LingMirror agent value should emphasize transparent diagnosis and one-click-but-audited actions. |
| BigSeller | Positions a free omnichannel solution for Shopee/Lazada/TikTok Shop and 20+ platforms, with product, order, inventory, purchase, marketing, distribution, POS, report, app, omnichannel, and sourcing modules. | official_site | https://www.bigseller.com/ | Maintain spreadsheet/batch-friendly workflows for SME adoption. |
| 4Seller | Presents multi-platform listing, order management, inventory management, Amazon MCF, shipping carriers, bulk edit, and Amazon profit calculator. | official_site | https://www.4seller.com/ | Use as a reference for simple listing/order/inventory surface and bulk operations. |
| Sellercloud | Presents catalog, inventory, warehouse, order management, order rule engine, purchasing, shipping, reporting, accounting, web service API, and 350+ integrations. | official_site | https://sellercloud.com/ | Use as an architecture benchmark for module boundaries and automation rules. |
| ShipStation | Presents carrier and marketplace integrations plus APIs for order consolidation, labels, rate comparison, address validation, checkout, tracking, and analytics. | official_site | https://www.shipstation.com/ | Shipping should become adapter-backed TMS, not scattered manual fields. |
| Linnworks | Presents order/shipping, inventory, warehouse, marketplace listings, analytics, bulk listing, order routing, carrier labels, tracking, and cost control. | official_site | https://www.linnworks.com/ | Add automation after canonical order/shipping/inventory records exist. |

## Research Confidence Notes

- Mabang, SellFox, Dianxiaomi, Eccang, Tongtool, CaptainBI, BigSeller, 4Seller, Sellercloud, ShipStation, and Linnworks are supported by official public pages.
- Some vendor counts such as platform, logistics, warehouse, and user totals are marketing claims. Use them as evidence of positioning and ecosystem ambition, not verified operational quality.
- Help-center depth was not exhaustively reviewed in this pass. When implementing a specific integration, run a second source pass on official API/help docs.
