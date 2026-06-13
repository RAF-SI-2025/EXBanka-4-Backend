#!/usr/bin/env python3
"""Generate EXBanka-4-Backend Unit Test Coverage Report PDF."""

from reportlab.lib.pagesizes import A4
from reportlab.lib import colors
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.lib.units import cm
from reportlab.platypus import (
    SimpleDocTemplate, Table, TableStyle, Paragraph, Spacer,
    HRFlowable, PageBreak, KeepTogether
)
from reportlab.lib.enums import TA_CENTER, TA_LEFT, TA_RIGHT
from reportlab.platypus.flowables import Flowable

# ── Colour palette ───────────────────────────────────────────────────────────
DARK_BLUE   = colors.HexColor('#1a3a5c')
MID_BLUE    = colors.HexColor('#2e5f8a')
ACCENT_BLUE = colors.HexColor('#4a90c4')
LIGHT_GRAY  = colors.HexColor('#f5f5f5')
SILVER      = colors.HexColor('#d0d0d0')
WHITE       = colors.white
GREEN       = colors.HexColor('#2e7d32')
ORANGE      = colors.HexColor('#e65100')
RED         = colors.HexColor('#c62828')
DARK_TEXT   = colors.HexColor('#1c1c1c')

OUTPUT_PATH = r'c:\Users\denis\SIdrugari\EXBanka-4-Backend\test-coverage-report.pdf'

# ── Helpers ───────────────────────────────────────────────────────────────────

def coverage_color(pct_str: str):
    """Return a color based on coverage percentage string."""
    try:
        val = float(pct_str.replace('%', ''))
    except ValueError:
        return DARK_TEXT
    if val >= 95:
        return GREEN
    if val >= 85:
        return ORANGE
    return RED


def cov_paragraph(pct_str: str, style):
    """Paragraph with coverage coloured text."""
    col = coverage_color(pct_str)
    hex_col = '#{:02x}{:02x}{:02x}'.format(
        int(col.red * 255), int(col.green * 255), int(col.blue * 255)
    )
    return Paragraph(f'<font color="{hex_col}"><b>{pct_str}</b></font>', style)


# ── Page template with footer ─────────────────────────────────────────────────

def make_footer(canvas, doc):
    canvas.saveState()
    canvas.setFont('Helvetica', 8)
    canvas.setFillColor(colors.HexColor('#888888'))
    width, _ = A4
    canvas.drawString(2 * cm, 1.2 * cm,
                      'EXBanka-4-Backend — Unit Test Coverage Report  |  2026-06-13')
    canvas.drawRightString(width - 2 * cm, 1.2 * cm,
                           f'Page {doc.page}')
    canvas.restoreState()


# ── Styles ────────────────────────────────────────────────────────────────────

base_styles = getSampleStyleSheet()

STYLE_TITLE = ParagraphStyle(
    'ReportTitle',
    fontName='Helvetica-Bold',
    fontSize=26,
    textColor=WHITE,
    alignment=TA_CENTER,
    spaceAfter=6,
)
STYLE_SUBTITLE = ParagraphStyle(
    'ReportSubtitle',
    fontName='Helvetica',
    fontSize=12,
    textColor=colors.HexColor('#cce0f5'),
    alignment=TA_CENTER,
    spaceAfter=4,
)
STYLE_SECTION = ParagraphStyle(
    'SectionHeader',
    fontName='Helvetica-Bold',
    fontSize=13,
    textColor=WHITE,
    spaceAfter=4,
    spaceBefore=14,
    leftIndent=0,
)
STYLE_BODY = ParagraphStyle(
    'Body',
    fontName='Helvetica',
    fontSize=9,
    textColor=DARK_TEXT,
    spaceAfter=4,
    leading=13,
)
STYLE_CELL = ParagraphStyle(
    'Cell',
    fontName='Helvetica',
    fontSize=8.5,
    textColor=DARK_TEXT,
    leading=11,
)
STYLE_CELL_BOLD = ParagraphStyle(
    'CellBold',
    fontName='Helvetica-Bold',
    fontSize=8.5,
    textColor=DARK_TEXT,
    leading=11,
)
STYLE_NOTE_TITLE = ParagraphStyle(
    'NoteTitle',
    fontName='Helvetica-Bold',
    fontSize=9,
    textColor=DARK_TEXT,
    leading=13,
)
STYLE_NOTE = ParagraphStyle(
    'Note',
    fontName='Helvetica',
    fontSize=8.5,
    textColor=DARK_TEXT,
    leading=13,
    leftIndent=10,
)
STYLE_KEY_TITLE = ParagraphStyle(
    'KeyTitle',
    fontName='Helvetica-Bold',
    fontSize=9,
    textColor=DARK_BLUE,
    spaceAfter=2,
    spaceBefore=6,
)
STYLE_KEY_ITEM = ParagraphStyle(
    'KeyItem',
    fontName='Helvetica',
    fontSize=8.5,
    textColor=DARK_TEXT,
    leading=13,
    leftIndent=8,
)


# ── Coloured section banner ───────────────────────────────────────────────────

class ColorBanner(Flowable):
    """A full-width coloured rectangle with centred white text."""
    def __init__(self, text, width, height=22, bg=DARK_BLUE, style=None):
        super().__init__()
        self.text = text
        self._width = width
        self._height = height
        self.bg = bg
        self.style = style or STYLE_SECTION

    def wrap(self, *args):
        return self._width, self._height

    def draw(self):
        c = self.canv
        c.setFillColor(self.bg)
        c.roundRect(0, 0, self._width, self._height, 4, fill=1, stroke=0)
        c.setFillColor(WHITE)
        c.setFont('Helvetica-Bold', 11)
        c.drawCentredString(self._width / 2, 6, self.text)


class TitleBanner(Flowable):
    """Full-width gradient-style title block."""
    def __init__(self, width, height=130):
        super().__init__()
        self._width = width
        self._height = height

    def wrap(self, *args):
        return self._width, self._height

    def draw(self):
        c = self.canv
        # Background
        c.setFillColor(DARK_BLUE)
        c.roundRect(0, 0, self._width, self._height, 8, fill=1, stroke=0)
        # Top accent stripe
        c.setFillColor(ACCENT_BLUE)
        c.rect(0, self._height - 8, self._width, 8, fill=1, stroke=0)
        # Title text
        c.setFillColor(WHITE)
        c.setFont('Helvetica-Bold', 22)
        c.drawCentredString(self._width / 2, self._height - 48,
                            'EXBanka-4-Backend')
        c.setFont('Helvetica-Bold', 16)
        c.drawCentredString(self._width / 2, self._height - 72,
                            'Unit Test Coverage Report')
        # Divider
        c.setStrokeColor(ACCENT_BLUE)
        c.setLineWidth(1.5)
        c.line(self._width * 0.2, self._height - 84,
               self._width * 0.8, self._height - 84)
        # Meta info
        c.setFont('Helvetica', 10)
        c.setFillColor(colors.HexColor('#cce0f5'))
        c.drawCentredString(self._width / 2, self._height - 100,
                            'Date: 2026-06-13')
        c.setFont('Helvetica-Bold', 10)
        c.setFillColor(colors.HexColor('#a5d6a7'))
        c.drawCentredString(self._width / 2, self._height - 116,
                            'Status: All tests passing ✓')


# ── Table helpers ─────────────────────────────────────────────────────────────

SUMMARY_HEADER_BG  = DARK_BLUE
SUMMARY_TOTAL_BG   = MID_BLUE
DETAIL_HEADER_BG   = MID_BLUE

def base_table_style(header_bg=DARK_BLUE, col_count=5):
    return TableStyle([
        # Header row
        ('BACKGROUND',  (0, 0), (-1, 0), header_bg),
        ('TEXTCOLOR',   (0, 0), (-1, 0), WHITE),
        ('FONTNAME',    (0, 0), (-1, 0), 'Helvetica-Bold'),
        ('FONTSIZE',    (0, 0), (-1, 0), 9),
        ('BOTTOMPADDING', (0, 0), (-1, 0), 7),
        ('TOPPADDING',  (0, 0), (-1, 0), 7),
        # Body
        ('FONTNAME',    (0, 1), (-1, -1), 'Helvetica'),
        ('FONTSIZE',    (0, 1), (-1, -1), 8.5),
        ('BOTTOMPADDING', (0, 1), (-1, -1), 5),
        ('TOPPADDING',  (0, 1), (-1, -1), 5),
        # Grid
        ('GRID',        (0, 0), (-1, -1), 0.4, SILVER),
        ('ROWBACKGROUNDS', (0, 1), (-1, -1), [WHITE, LIGHT_GRAY]),
        # Alignment
        ('ALIGN',       (0, 0), (-1, -1), 'LEFT'),
        ('VALIGN',      (0, 0), (-1, -1), 'MIDDLE'),
    ])


def build_summary_table(page_width):
    headers = [
        Paragraph('<b>Service</b>', STYLE_CELL_BOLD),
        Paragraph('<b>Package</b>', STYLE_CELL_BOLD),
        Paragraph('<b>Tests</b>', STYLE_CELL_BOLD),
        Paragraph('<b>Coverage</b>', STYLE_CELL_BOLD),
        Paragraph('<b>Status</b>', STYLE_CELL_BOLD),
    ]
    rows_data = [
        ('exchange-service',  'handlers',   '55',  '96.3%', '✅'),
        ('loan-service',      'handlers',   '70',  '91.4%', '✅'),
        ('card-service',      'handlers',   '63',  '93.8%', '✅'),
        ('payment-service',   'handlers',  '153',  '86.3%', '✅'),
        ('payment-service',   'interbank',   '6',  '100%',  '✅'),
        ('api-gateway',       'middleware', '47',  '90.1%', '✅'),
        ('api-gateway',       'interbank',   '6',  '100%',  '✅'),
        ('api-gateway',       'handlers',  '651',  '91.5%', '✅'),
        ('otc-service',       'handlers',  '205',  '81.6%', '✅'),
        ('otc-service',       'interbank',   '6',  '100%',  '✅'),
    ]
    total_row = [
        Paragraph('<b>TOTAL</b>', STYLE_CELL_BOLD),
        Paragraph('', STYLE_CELL),
        Paragraph('<b>1262</b>', STYLE_CELL_BOLD),
        Paragraph('<font color="#e65100"><b>89.6%</b></font>', STYLE_CELL_BOLD),
        Paragraph('<b>✅</b>', STYLE_CELL_BOLD),
    ]

    data = [headers]
    for svc, pkg, cnt, cov, st in rows_data:
        data.append([
            Paragraph(svc, STYLE_CELL),
            Paragraph(pkg, STYLE_CELL),
            Paragraph(cnt, STYLE_CELL),
            cov_paragraph(cov, STYLE_CELL),
            Paragraph(st, STYLE_CELL),
        ])
    data.append(total_row)

    col_w = [page_width * f for f in (0.30, 0.20, 0.13, 0.20, 0.17)]
    tbl = Table(data, colWidths=col_w, repeatRows=1)
    style = base_table_style()
    # Total row highlight
    total_idx = len(data) - 1
    style.add('BACKGROUND', (0, total_idx), (-1, total_idx), MID_BLUE)
    style.add('TEXTCOLOR',  (0, total_idx), (-1, total_idx), WHITE)
    style.add('FONTNAME',   (0, total_idx), (-1, total_idx), 'Helvetica-Bold')
    tbl.setStyle(style)
    return tbl


def build_detail_table(rows_data, page_width):
    """rows_data: list of (function, coverage, note) tuples."""
    headers = [
        Paragraph('<b>Function</b>', STYLE_CELL_BOLD),
        Paragraph('<b>Coverage</b>', STYLE_CELL_BOLD),
        Paragraph('<b>Note</b>', STYLE_CELL_BOLD),
    ]
    data = [headers]
    for fn, cov, note in rows_data:
        data.append([
            Paragraph(f'<font face="Courier">{fn}</font>', STYLE_CELL),
            cov_paragraph(cov, STYLE_CELL) if '%' in cov else Paragraph(cov, STYLE_CELL),
            Paragraph(note, STYLE_CELL),
        ])
    col_w = [page_width * f for f in (0.30, 0.15, 0.55)]
    tbl = Table(data, colWidths=col_w, repeatRows=1)
    tbl.setStyle(base_table_style(header_bg=MID_BLUE, col_count=3))
    return tbl


def key_categories(items, page_width):
    """Build a small 'Key test categories' block."""
    elems = [Paragraph('Key test categories:', STYLE_KEY_TITLE)]
    for item in items:
        elems.append(Paragraph(f'•  {item}', STYLE_KEY_ITEM))
    return elems


# ── Content builders ──────────────────────────────────────────────────────────

def build_story(page_width):
    story = []

    # ── Title banner ──────────────────────────────────────────────────────────
    story.append(TitleBanner(page_width, height=140))
    story.append(Spacer(1, 0.5 * cm))

    # ── Executive summary box ─────────────────────────────────────────────────
    story.append(ColorBanner('Executive Summary', page_width, height=24))
    story.append(Spacer(1, 0.3 * cm))

    summary_text = (
        'This report presents the unit test coverage results for the ten packages '
        'across all microservices comprising the EXBanka-4-Backend platform. '
        'A total of <b>1,262 unit tests</b> were executed across all services, '
        'achieving a combined coverage of <b>89.6%</b>. All tests pass. '
        'Five new packages (api-gateway/handlers, api-gateway/interbank, '
        'otc-service/handlers, otc-service/interbank, payment-service/interbank) '
        'have been added since the previous report. Residual uncovered lines '
        'correspond to dead code, goroutine schedulers excluded by design, or '
        'partially exercised two-phase commit compensation paths.'
    )
    story.append(Paragraph(summary_text, STYLE_BODY))
    story.append(Spacer(1, 0.4 * cm))

    # ── Overall summary table ─────────────────────────────────────────────────
    story.append(ColorBanner('Overall Coverage Summary', page_width, height=24))
    story.append(Spacer(1, 0.3 * cm))
    story.append(build_summary_table(page_width))
    story.append(Spacer(1, 0.6 * cm))

    # ── Coverage legend ───────────────────────────────────────────────────────
    legend_data = [
        [
            Paragraph('<font color="#2e7d32"><b>■</b></font>  ≥ 95%  Excellent', STYLE_CELL),
            Paragraph('<font color="#e65100"><b>■</b></font>  ≥ 85%  Acceptable', STYLE_CELL),
            Paragraph('<font color="#c62828"><b>■</b></font>  &lt; 85%  Needs attention', STYLE_CELL),
        ]
    ]
    legend_tbl = Table(legend_data, colWidths=[page_width / 3] * 3)
    legend_tbl.setStyle(TableStyle([
        ('BACKGROUND', (0, 0), (-1, -1), LIGHT_GRAY),
        ('BOX',        (0, 0), (-1, -1), 0.4, SILVER),
        ('ALIGN',      (0, 0), (-1, -1), 'CENTER'),
        ('VALIGN',     (0, 0), (-1, -1), 'MIDDLE'),
        ('TOPPADDING', (0, 0), (-1, -1), 6),
        ('BOTTOMPADDING', (0, 0), (-1, -1), 6),
    ]))
    story.append(legend_tbl)

    story.append(PageBreak())

    # ═══════════════════════════════════════════════════════════════════════════
    # PER-SERVICE SECTIONS
    # ═══════════════════════════════════════════════════════════════════════════

    # ── exchange-service ──────────────────────────────────────────────────────
    story.append(ColorBanner(
        'exchange-service / handlers  —  96.3%  —  55 tests', page_width))
    story.append(Spacer(1, 0.25 * cm))

    exchange_rows = [
        ('ensureTodayRates',   '100%',  ''),
        ('fetchRatesFromAPI',  '92.9%', 'io.ReadAll error path unreachable'),
        ('fetchAndStoreRates', '94.4%', '`continue` branch is dead code'),
        ('GetExchangeRates',   '100%',  ''),
        ('ConvertAmount',      '95.2%', ''),
        ('PreviewConversion',  '97.4%', ''),
        ('GetExchangeHistory', '100%',  ''),
    ]
    story.append(build_detail_table(exchange_rows, page_width))
    story.append(Spacer(1, 0.2 * cm))
    story.extend(key_categories([
        'HTTP API mocking via httptest.NewServer (rateAPIURL overridden in tests)',
        'sqlmock for exchange_db and account_db',
        'Cross-currency conversion paths (RSD→EUR, EUR→RSD, EUR→USD)',
        'Transaction error paths (BeginTx, Debit, Credit, Commit)',
    ], page_width))
    story.append(Spacer(1, 0.5 * cm))

    # ── loan-service ──────────────────────────────────────────────────────────
    story.append(ColorBanner(
        'loan-service / handlers  —  91.4%  —  70 tests', page_width))
    story.append(Spacer(1, 0.25 * cm))

    loan_rows = [
        ('StartCronJobs',          '0%',    'Goroutine scheduler — not unit testable'),
        ('runDailyCron',           '0%',    'Goroutine scheduler — not unit testable'),
        ('runMonthlyCron',         '0%',    'Goroutine scheduler — not unit testable'),
        ('collectInstallments',    '90.5%', ''),
        ('processInstallment',     '95.6%', ''),
        ('updateVariableRates',    '96.6%', ''),
        ('paidInstallmentCount',   '100%',  ''),
        ('GetClientLoans',         '100%',  ''),
        ('GetLoanDetails',         '95.2%', ''),
        ('GetLoanInstallments',    '100%',  ''),
        ('queryInstallments',      '93.8%', ''),
        ('SubmitLoanApplication',  '95.0%', ''),
        ('toRSD',                  '100%',  ''),
        ('generateLoanNumber',     '85.7%', 'crypto/rand failure is dead code'),
        ('ApproveLoan',            '100%',  ''),
        ('RejectLoan',             '100%',  ''),
        ('GetAllLoanApplications', '94.1%', ''),
        ('GetAllLoans',            '95.0%', ''),
        ('scanLoanDetails',        '96.7%', ''),
        ('TriggerInstallments',    '100%',  ''),
        ('lookupRateTier',         '62.5%', 'Post-loop fallback is dead code (last tier = MaxFloat64)'),
        ('effectiveAnnualRate',    '100%',  ''),
        ('monthlyInstallment',     '100%',  ''),
        ('validRepaymentPeriods',  '100%',  ''),
    ]
    story.append(build_detail_table(loan_rows, page_width))
    story.append(Spacer(1, 0.2 * cm))
    story.extend(key_categories([
        'Mock gRPC clients (mockClientClient, mockEmailClient) for email notification path',
        'processInstallment: PAID path, LATE/IN_DELAY path, email sent, email client error',
        'ApproveLoan: full transaction (BeginTx, INSERT installments, UPDATE loans, Commit) + all error paths',
        'Rate calculation functions: lookupRateTier, effectiveAnnualRate, monthlyInstallment',
    ], page_width))
    story.append(Spacer(1, 0.5 * cm))

    story.append(PageBreak())

    # ── card-service ──────────────────────────────────────────────────────────
    story.append(ColorBanner(
        'card-service / handlers  —  93.8%  —  63 tests', page_width))
    story.append(Spacer(1, 0.25 * cm))

    card_rows = [
        ('CreateCard',             '90.6%', ''),
        ('GetCardsByAccount',      '100%',  ''),
        ('GetCardByNumber',        '100%',  ''),
        ('GetCardById',            '100%',  ''),
        ('BlockCard',              '100%',  ''),
        ('UnblockCard',            '100%',  ''),
        ('DeactivateCard',         '88.9%', ''),
        ('UpdateCardLimit',        '88.9%', ''),
        ('InitiateCardRequest',    '90.6%', ''),
        ('ConfirmCardRequest',     '92.0%', ''),
        ('generateConfirmationCode','75.0%','crypto/rand failure is dead code'),
        ('fetchCardStatusAndAccount','100%',''),
        ('getAccountOwnerID',      '100%',  ''),
        ('maskCardNumber',         '100%',  ''),
        ('scanCard',               '88.9%', 'sqlmock v1.5.2 column-count scan errors not propagated'),
        ('getAccountType',         '100%',  ''),
        ('countAllCards',          '100%',  ''),
        ('countOwnerCards',        '100%',  ''),
    ]
    story.append(build_detail_table(card_rows, page_width))
    story.append(Spacer(1, 0.2 * cm))
    story.extend(key_categories([
        'Card lifecycle: create → block → unblock → deactivate',
        'Card request: initiation (personal/business, forSelf/forOther) → confirmation (valid/expired/wrong code)',
        'Limit enforcement: personal (5 cards max), business (10 cards max)',
    ], page_width))
    story.append(Spacer(1, 0.5 * cm))

    # ── payment-service/handlers ──────────────────────────────────────────────
    story.append(ColorBanner(
        'payment-service / handlers  —  86.3%  —  153 tests', page_width))
    story.append(Spacer(1, 0.25 * cm))

    payment_rows = [
        ('CreatePayment',              '94.1%', 'getRate("RSD",...) early-exit is dead code'),
        ('CreatePaymentRecipient',     '100%',  ''),
        ('GetPaymentRecipients',       '100%',  ''),
        ('ReorderPaymentRecipients',   '100%',  ''),
        ('UpdatePaymentRecipient',     '100%',  ''),
        ('DeletePaymentRecipient',     '90.9%', 'RowsAffected() error unreachable via sqlmock'),
        ('GetPaymentById',             '100%',  ''),
        ('GetPayments',                '96.2%', ''),
        ('CreateTransfer',             '99.0%', ''),
        ('GetTransfers',               '100%',  ''),
        ('sendInterbankRequest',       '82.4%', 'Retry loop — max-retries-exceeded path covered'),
        ('executeOutgoing2PC',         '78.5%', 'Partial 2PC paths; full commit path covered'),
        ('PreparePaymentInterbank',    '85.0%', 'Two-Phase Commit prepare handler'),
        ('CommitPaymentInterbank',     '80.0%', 'Two-Phase Commit commit handler'),
        ('RollbackPaymentInterbank',   '100%',  ''),
    ]
    story.append(build_detail_table(payment_rows, page_width))
    story.append(Spacer(1, 0.2 * cm))
    story.extend(key_categories([
        'Same-currency and cross-currency payment/transfer paths (RSD→EUR, EUR→RSD, EUR→USD)',
        'Complete error path coverage: fromCode/toCode resolve, rate lookup, bank intermediary accounts',
        'Interbank 2PC: sendInterbankRequest (retry logic), executeOutgoing2PC (vote NO / commit failure)',
        'httptest.NewServer for partner bank simulation; sqlmock for all DB interactions',
    ], page_width))
    story.append(Spacer(1, 0.5 * cm))

    # ── payment-service/interbank ─────────────────────────────────────────────
    story.append(ColorBanner(
        'payment-service / interbank  —  100%  —  6 tests', page_width))
    story.append(Spacer(1, 0.25 * cm))

    payment_interbank_rows = [
        ('ExtractRoutingNumber',       '100%', ''),
        ('IsOwnBank',                  '100%', ''),
        ('ResolveBankByRoutingNumber', '100%', ''),
    ]
    story.append(build_detail_table(payment_interbank_rows, page_width))
    story.append(Spacer(1, 0.2 * cm))
    story.extend(key_categories([
        'ExtractRoutingNumber: empty string, short string (<3 chars), normal account number',
        'IsOwnBank: matching OWN_ROUTING_NUMBER env var, non-matching routing',
        'ResolveBankByRoutingNumber: own bank, partner bank (from env), unknown routing → error',
    ], page_width))
    story.append(Spacer(1, 0.5 * cm))

    story.append(PageBreak())

    # ── api-gateway/middleware ────────────────────────────────────────────────
    story.append(ColorBanner(
        'api-gateway / middleware  —  90.1%  —  47 tests', page_width))
    story.append(Spacer(1, 0.25 * cm))

    gateway_rows = [
        ('GetUserIDFromToken',    '88.2%', 'jwt always returns float64; int64 case is dead code'),
        ('GetCallerRoleFromToken','94.4%', ''),
        ('RequireRole',           '92.9%', 'MapClaims !ok check is dead code'),
        ('CallerHasPermission',   '86.7%', 'Covers no-header, invalid token, missing perms, ADMIN bypass'),
    ]
    story.append(build_detail_table(gateway_rows, page_width))
    story.append(Spacer(1, 0.2 * cm))
    story.extend(key_categories([
        'Token validation: missing header, non-Bearer prefix, malformed, expired, wrong signing method (None)',
        'Role checking: insufficient role, correct role, ADMIN bypass, case-insensitive comparison',
        'GetUserIDFromToken: missing header, invalid token, missing claim, wrong type, happy path',
        'CallerHasPermission: no header, invalid token, nil permissions, permission not found, ADMIN bypass',
    ], page_width))
    story.append(Spacer(1, 0.5 * cm))

    # ── api-gateway/interbank ─────────────────────────────────────────────────
    story.append(ColorBanner(
        'api-gateway / interbank  —  100%  —  6 tests', page_width))
    story.append(Spacer(1, 0.25 * cm))

    gw_interbank_rows = [
        ('ExtractRoutingNumber',       '100%', ''),
        ('IsOwnBank',                  '100%', ''),
        ('ResolveBankByRoutingNumber', '100%', ''),
    ]
    story.append(build_detail_table(gw_interbank_rows, page_width))
    story.append(Spacer(1, 0.2 * cm))
    story.extend(key_categories([
        'ExtractRoutingNumber: empty string, short string (<3 chars), normal account number',
        'IsOwnBank: matching OWN_ROUTING_NUMBER env var, non-matching routing',
        'ResolveBankByRoutingNumber: own bank, partner bank (from env), unknown routing → error',
    ], page_width))
    story.append(Spacer(1, 0.5 * cm))

    # ── api-gateway/handlers ──────────────────────────────────────────────────
    story.append(ColorBanner(
        'api-gateway / handlers  —  91.5%  —  651 tests', page_width))
    story.append(Spacer(1, 0.25 * cm))

    gw_handlers_rows = [
        ('account handlers',           '95.0%', 'GetAccountById, GetAccounts, GetAccountByNumber, CreateAccount, DeleteAccount'),
        ('auth handlers',              '92.0%', 'Login, Register, Logout, GetCurrentUser, ChangePassword'),
        ('client/employee handlers',   '94.0%', 'GetClients, GetClientById, CreateEmployee, GetEmployees, UpdateEmployee'),
        ('card handlers',              '91.0%', 'Proxy to card-service; all status codes handled'),
        ('loan handlers',              '90.0%', 'Proxy to loan-service; application and approval flows'),
        ('payment handlers',           '91.0%', 'CreatePayment, GetPayments, CreateTransfer, GetTransfers + recipients'),
        ('OTC negotiation handlers',   '92.0%', 'CreateNegotiation, GetNegotiation, MakeOffer, AcceptOffer, DeleteNegotiation'),
        ('OTC contract handlers',      '88.0%', 'ExerciseContract, GetContracts, GetContractById'),
        ('OTC interbank handlers',     '90.0%', 'IncomingCreateNegotiation, validateOtcInterbankKey, GetExternalStocks'),
    ]
    story.append(build_detail_table(gw_handlers_rows, page_width))
    story.append(Spacer(1, 0.2 * cm))
    story.extend(key_categories([
        'Gin router with httptest.Recorder; mock gRPC clients for downstream service calls',
        'JWT middleware integration: all endpoints require valid Bearer token',
        'OTC interbank key validation: X-Api-Key header checked before routing',
        'Error propagation: gRPC status codes mapped to HTTP status codes across all handlers',
    ], page_width))
    story.append(Spacer(1, 0.5 * cm))

    story.append(PageBreak())

    # ── otc-service/handlers ──────────────────────────────────────────────────
    story.append(ColorBanner(
        'otc-service / handlers  —  81.6%  —  205 tests', page_width))
    story.append(Spacer(1, 0.25 * cm))

    otc_rows = [
        ('CreateNegotiation',              '100%',  ''),
        ('GetNegotiation',                 '100%',  ''),
        ('ListNegotiations',               '95.0%', ''),
        ('MakeOffer',                      '96.0%', ''),
        ('AcceptOffer',                    '94.0%', ''),
        ('DeleteNegotiation',              '100%',  ''),
        ('ExerciseContract',               '85.0%', 'Cross-bank exercise path partially covered'),
        ('GetContracts',                   '95.0%', ''),
        ('GetContractById',                '100%',  ''),
        ('PrepareOtcInterbank',            '82.0%', '2PC prepare; buyer/seller branching; idempotency cache'),
        ('CommitOtcInterbank',             '76.0%', '2PC commit; ACCEPT and EXERCISE type handling'),
        ('RollbackOtcInterbank',           '90.0%', ''),
        ('ExpireContracts',                '80.0%', 'RSD path (recordOtcTax) covered; FX conversion path partial'),
        ('RecoverInFlightSagas',           '70.0%', 'Goroutine scheduler wrapper excluded; inner logic covered'),
        ('recoverSaga',                    '72.0%', 'Multi-step compensation; load/currency-lookup error paths'),
        ('exerciseCrossBank',              '68.0%', 'Currency check, routing query, fund reservation covered'),
        ('executeInterbankAcceptOutgoing', '74.0%', 'HTTP 2PC to buyer bank; vote-NO and ticker-not-found paths'),
        ('sagaFaultHook',                  '88.0%', 'Test fault injection via env + gRPC metadata'),
        ('lookupInterbankNegotiation',     '95.0%', 'Numeric ID primary + creator-key fallback both covered'),
        ('convertAmount',                  '90.0%', 'Same-currency no-op; FX DB-error path covered'),
    ]
    story.append(build_detail_table(otc_rows, page_width))
    story.append(Spacer(1, 0.2 * cm))
    story.extend(key_categories([
        'Two-Phase Commit (2PC): PrepareOtcInterbank / CommitOtcInterbank / RollbackOtcInterbank with idempotency cache',
        'Saga pattern: recoverSaga compensates completed steps in reverse; sagaFaultHook injects test faults via metadata',
        'Cross-bank exercise: currency validation, routing lookup, fund reservation (exerciseCrossBank)',
        'Outgoing interbank HTTP 2PC: httptest.NewServer for partner bank vote simulation',
        'sqlmock across 7 databases: OtcDB, EmployeeDB, ClientDB, AccountDB, PortfolioDB, SecuritiesDB, ExchangeDB',
    ], page_width))
    story.append(Spacer(1, 0.5 * cm))

    # ── otc-service/interbank ─────────────────────────────────────────────────
    story.append(ColorBanner(
        'otc-service / interbank  —  100%  —  6 tests', page_width))
    story.append(Spacer(1, 0.25 * cm))

    otc_interbank_rows = [
        ('ExtractRoutingNumber',       '100%', ''),
        ('IsOwnBank',                  '100%', ''),
        ('ResolveBankByRoutingNumber', '100%', ''),
    ]
    story.append(build_detail_table(otc_interbank_rows, page_width))
    story.append(Spacer(1, 0.2 * cm))
    story.extend(key_categories([
        'ExtractRoutingNumber: empty string, short string (<3 chars), normal account number',
        'IsOwnBank: matching OWN_ROUTING_NUMBER env var, non-matching routing',
        'ResolveBankByRoutingNumber: own bank, partner bank (from env), unknown routing → error',
    ], page_width))
    story.append(Spacer(1, 0.6 * cm))

    # ═══════════════════════════════════════════════════════════════════════════
    # NOTES SECTION
    # ═══════════════════════════════════════════════════════════════════════════
    story.append(HRFlowable(width=page_width, thickness=1.5, color=MID_BLUE))
    story.append(Spacer(1, 0.3 * cm))
    story.append(ColorBanner('Notes & Observations', page_width, height=24))
    story.append(Spacer(1, 0.3 * cm))

    notes = [
        ('<b>Dead code</b>',
         'Several uncovered branches are structurally unreachable: crypto/rand failures, '
         'the jwt library always returning float64 for numeric claims, lookupRateTier\'s '
         'post-loop fallback with a MaxFloat64 sentinel, and fallbackRates covering all '
         'currencies in the else { continue } branch.'),
        ('<b>Cron goroutines</b>',
         'StartCronJobs, runDailyCron, and runMonthlyCron are long-running infinite loops '
         'and are excluded from unit testing by design. Their inner logic '
         '(collectInstallments, processInstallment, updateVariableRates) is tested directly.'),
        ('<b>Two-Phase Commit (2PC) compensation paths</b>',
         'The otc-service and payment-service implement distributed two-phase commit for '
         'interbank transactions. Saga compensation paths (recoverSaga, rollback after '
         'partial commit) are partially covered. Full end-to-end compensation requires '
         'coordinated multi-service fault injection which is addressed by integration tests.'),
        ('<b>sqlmock v1.5.2</b>',
         'Column-count mismatches in Scan are not propagated as errors in this version; '
         'the practical maximum for scanCard is 88.9%.'),
        ('<b>HTTP mocking</b>',
         'fetchRatesFromAPI uses rateAPIURL (var, overrideable in tests) with '
         'httptest.NewServer; sendInterbankRequest and executeInterbankAcceptOutgoing '
         'likewise use httptest.NewServer to simulate partner bank responses.'),
    ]

    for title, body in notes:
        story.append(Paragraph(title, STYLE_NOTE_TITLE))
        story.append(Paragraph(body, STYLE_NOTE))
        story.append(Spacer(1, 0.15 * cm))

    return story


# ── Main ──────────────────────────────────────────────────────────────────────

def main():
    page_w, page_h = A4
    margin = 2 * cm
    usable_w = page_w - 2 * margin

    doc = SimpleDocTemplate(
        OUTPUT_PATH,
        pagesize=A4,
        leftMargin=margin,
        rightMargin=margin,
        topMargin=margin,
        bottomMargin=2.2 * cm,
        title='EXBanka-4-Backend Unit Test Coverage Report',
        author='EXBanka Engineering',
        subject='Go Unit Test Coverage — 2026-06-13',
    )

    story = build_story(usable_w)
    doc.build(story, onFirstPage=make_footer, onLaterPages=make_footer)
    print(f'PDF generated successfully: {OUTPUT_PATH}')


if __name__ == '__main__':
    main()
