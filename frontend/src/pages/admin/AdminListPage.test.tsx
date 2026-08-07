import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError } from "../../api/client";
import { AdminListPage } from "./AdminListPage";

type Item = { id: string; name: string };
const loadMock = vi.fn();

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/admin?keyword=initial&page=1"]}>
      <AdminListPage<Item>
        title="Test管理"
        load={loadMock}
        columns={[{ label: "名前", render: (item) => item.name }]}
        rowKey={(item) => item.id}
        actions={(item, reload) => <button type="button" onClick={reload}>{item.name}を再読込</button>}
      />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  loadMock.mockReset();
});

describe("AdminListPage", () => {
  it("loads, searches and paginates", async () => {
    loadMock
      .mockResolvedValueOnce({ items: [{ id: "1", name: "Alpha" }], page: 1, total_pages: 2 })
      .mockResolvedValueOnce({ items: [{ id: "2", name: "Beta" }], page: 1, total_pages: 2 })
      .mockResolvedValueOnce({ items: [], page: 2, total_pages: 2 });
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("Alpha")).toBeInTheDocument();
    expect(loadMock).toHaveBeenCalledWith("initial", 1);

    const keyword = screen.getByLabelText("キーワード");
    await user.clear(keyword);
    await user.type(keyword, "beta");
    await user.click(screen.getByRole("button", { name: "検索" }));
    await waitFor(() => expect(loadMock).toHaveBeenCalledWith("beta", 1));
    expect(await screen.findByText("Beta")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "次へ" }));
    await waitFor(() => expect(loadMock).toHaveBeenCalledWith("beta", 2));
    expect(await screen.findByText("対象データはありません")).toBeInTheDocument();
  });

  it("reload action executes the current query again", async () => {
    loadMock.mockResolvedValue({ items: [{ id: "1", name: "Alpha" }], page: 1, total_pages: 1 });
    const user = userEvent.setup();
    renderPage();
    await screen.findByText("Alpha");

    await user.click(screen.getByRole("button", { name: "Alphaを再読込" }));
    await waitFor(() => expect(loadMock).toHaveBeenCalledTimes(2));
  });

  it("shows API errors", async () => {
    loadMock.mockRejectedValue(new ApiRequestError({ status: 500, code: "INTERNAL_ERROR", message: "管理APIエラー", request_id: "req-admin" }));
    renderPage();
    expect(await screen.findByRole("alert")).toHaveTextContent("管理APIエラー");
  });
});
