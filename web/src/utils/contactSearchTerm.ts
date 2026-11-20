// 联系人列表的搜索参数。
//
// 只要输入不是空白就要把搜索词发给后端。以前各处写的是 search.length > 2，
// 这是按拉丁字母前缀定的阈值：一到两个字的姓名（多数中文、日文、韩文姓名）
// 永远搜不出来，输入框却没有任何提示。后端是 LIKE 匹配，对短词同样有效。
export function contactSearchParam(term: string): { search?: string } {
  const trimmed = term.trim();
  return trimmed === "" ? {} : { search: trimmed };
}
