import ExcelJS from 'exceljs';
import type { Timesheet } from './timesheets';
import { exportTimestamp } from './export-filename';

export async function createTimesheetWorkbook(rows: readonly Timesheet[]) {
  const workbook = new ExcelJS.Workbook();
  const sheet = workbook.addWorksheet('Timesheets', {
    views: [{ state: 'frozen', ySplit: 1 }],
  });
  sheet.columns = [
    { header: 'Work date', key: 'date', width: 18 },
    { header: 'Description', key: 'description', width: 60 },
    { header: 'Hours', key: 'hours', width: 12 },
    { header: 'Status', key: 'status', width: 16 },
  ];
  for (const row of rows) {
    sheet.addRow({
      date: new Date(`${row.work_date}T00:00:00Z`),
      // Keep descriptions as text, including values beginning with '='.
      description: row.description ?? '',
      hours: row.hours_worked,
      status: row.status,
    });
  }
  sheet.getColumn('date').numFmt = 'dd mmm yyyy';
  sheet.getColumn('hours').numFmt = '0';
  sheet.getColumn('description').alignment = { wrapText: true, vertical: 'top' };
  sheet.getRow(1).font = { bold: true };
  sheet.autoFilter = { from: 'A1', to: `D${rows.length + 1}` };
  const totalRow = sheet.addRow({ description: 'TOTAL' });
  totalRow.getCell('hours').value = rows.length ? {
    formula: `SUM(C2:C${rows.length + 1})`,
    result: rows.reduce((total, row) => total + row.hours_worked, 0),
  } : 0;
  totalRow.font = { bold: true };
  totalRow.border = { top: { style: 'thin' } };
  const buffer = await workbook.xlsx.writeBuffer();
  return new Blob([new Uint8Array(buffer)], {
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  });
}

export async function downloadTimesheets(rows: readonly Timesheet[]) {
  const blob = await createTimesheetWorkbook(rows);
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `finos-timesheets-${exportTimestamp()}.xlsx`;
  document.body.appendChild(link);
  try {
    link.click();
  } finally {
    link.remove();
    // Give the browser time to start reading the download before releasing it.
    window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
  }
}
