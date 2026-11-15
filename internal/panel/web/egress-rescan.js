import {fa} from './core.js';

// The immutable reset cohort is distinct from today's changing source inventory.
// settled includes conclusive follow-ups; definitive counts first results only.
export function rescanSummary(run){
 if(!run?.id)return 'هنوز دور بازآزمایی کلی آغاز نشده است';
 return 'دور '+fa.format(run.id)+' · '+fa.format(run.completed)+' از '+fa.format(run.total)+' هویت یکتای این دور بررسی اولیه شده · '+fa.format(run.pending)+' در انتظار · '+fa.format(run.excluded)+' کنارگذاشته‌شده · '+fa.format(run.settled)+' تعیین تکلیف‌شده در این دور ('+fa.format(run.definitive)+' در اولین بررسی) · پذیرفته‌شده‌های قبلی: '+fa.format(run.accepted_completed)+' بررسی‌شده و '+fa.format(run.accepted_excluded||0)+' کنارگذاشته‌شده · '+(run.finished?'بررسی اولیهٔ دور تمام شد':'بررسی اولیه در حال اجرا')+(run.first_1000_ms?' · بررسی اولیهٔ ۱۰۰۰ هویت: '+fa.format(Math.round(run.first_1000_ms/1000))+' ثانیه':'')+(run.first_2000_ms?' · بررسی اولیهٔ ۲۰۰۰ هویت: '+fa.format(Math.round(run.first_2000_ms/1000))+' ثانیه':'');
}
