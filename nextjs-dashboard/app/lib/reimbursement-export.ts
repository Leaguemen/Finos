import ExcelJS from 'exceljs';
import type { Reimbursement } from './reimbursements';
import type { TimesheetOption } from './timesheets';

export async function createReimbursementWorkbook(
  rows: readonly Reimbursement[],
  employeeName: (userId: number) => string,
  timesheets: ReadonlyMap<number, TimesheetOption>,
  includeEmployee: boolean,
) {
  const workbook = new ExcelJS.Workbook();
  const sheet = workbook.addWorksheet('Reimbursements', { views: [{ state: 'frozen', ySplit: 1 }] });
  sheet.columns = [
    { header: 'Expense date', key: 'expenseDate', width: 18 },
    ...(includeEmployee ? [{ header: 'Employee', key: 'employee', width: 28 }] : []),
    { header: 'Timesheet', key: 'timesheet', width: 48 },
    { header: 'Category', key: 'category', width: 24 },
    { header: 'Description', key: 'description', width: 52 },
    { header: 'Amount', key: 'amount', width: 18 },
    { header: 'Status', key: 'status', width: 16 },
    { header: 'Receipt', key: 'receipt', width: 14 },
  ];
  for (const row of rows) {
    const timesheet = timesheets.get(row.timesheet_id);
    sheet.addRow({
      expenseDate: new Date(`${row.expense_date}T00:00:00Z`),
      ...(includeEmployee ? { employee: employeeName(row.user_id) } : {}),
      timesheet: timesheet?.description?.trim() || 'No description',
      // ExcelJS stores these values as text, including values beginning with '='.
      category: row.category,
      description: row.description ?? '',
      amount: Number(row.amount),
      status: row.status,
      receipt: row.has_receipt ? 'Yes' : 'No',
    });
  }
  sheet.getColumn('expenseDate').numFmt = 'dd mmm yyyy';
  sheet.getColumn('timesheet').alignment = { wrapText: true, vertical: 'top' };
  sheet.getColumn('amount').numFmt = '#,##0.00';
  sheet.getColumn('description').alignment = { wrapText: true, vertical: 'top' };
  sheet.getRow(1).font = { bold: true };
  sheet.autoFilter = { from: 'A1', to: `${String.fromCharCode(64 + sheet.columnCount)}${rows.length + 1}` };
  const buffer = await workbook.xlsx.writeBuffer();
  return new Blob([new Uint8Array(buffer)], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' });
}

export async function downloadReimbursements(
  rows: readonly Reimbursement[],
  date: string,
  employeeName: (userId: number) => string,
  timesheets: ReadonlyMap<number, TimesheetOption>,
  includeEmployee: boolean,
) {
  const blob = await createReimbursementWorkbook(rows, employeeName, timesheets, includeEmployee);
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `finos-reimbursements-${date}.xlsx`;
  document.body.appendChild(link);
  try { link.click(); } finally {
    link.remove();
    window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
  }
}
