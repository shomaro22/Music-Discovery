import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError } from "../api/client";
import { AuthProvider } from "../auth/AuthContext";
import { login } from "../api/user";
import { LoginPage } from "./LoginPage";

vi.mock("../api/user", () => ({ login: vi.fn(), signUp: vi.fn() }));
const mockedLogin = vi.mocked(login);
function renderPage(){return render(<MemoryRouter initialEntries={["/login"]}><AuthProvider><Routes><Route path="/login" element={<LoginPage/>}/><Route path="/" element={<p>TOP</p>}/></Routes></AuthProvider></MemoryRouter>)}
beforeEach(()=>{mockedLogin.mockReset();localStorage.clear()});
describe("LoginPage",()=>{
 it("renders initial fields",()=>{renderPage();expect(screen.getByRole("heading",{name:"ログイン"})).toBeInTheDocument();expect(screen.getByLabelText("メールアドレス")).toBeEnabled();expect(screen.getByLabelText("パスワード")).toHaveAttribute("type","password")});
 it("validates 7 and accepts 8 characters",async()=>{const user=userEvent.setup();renderPage();await user.type(screen.getByLabelText("メールアドレス"),"a@b.com");await user.type(screen.getByLabelText("パスワード"),"1234567");await user.click(screen.getByRole("button",{name:"ログイン"}));expect(screen.getByText("パスワードは8文字以上で入力してください")).toBeInTheDocument();expect(mockedLogin).not.toHaveBeenCalled()});
 it("rejects passwords over 72 bytes including multibyte",async()=>{const user=userEvent.setup();renderPage();await user.type(screen.getByLabelText("メールアドレス"),"a@b.com");await user.type(screen.getByLabelText("パスワード"),"あ".repeat(25));await user.click(screen.getByRole("button",{name:"ログイン"}));expect(screen.getByText("パスワードが長すぎます")).toBeInTheDocument()});
 it("normalizes email, preserves password, disables duplicate submission, and redirects",async()=>{let resolve!:(v:any)=>void;const promise=new Promise<any>(r=>resolve=r);mockedLogin.mockReturnValue(promise);const user=userEvent.setup();renderPage();await user.type(screen.getByLabelText("メールアドレス")," USER@EXAMPLE.COM ");await user.type(screen.getByLabelText("パスワード")," password ");await user.click(screen.getByRole("button",{name:"ログイン"}));expect(mockedLogin).toHaveBeenCalledWith({email:"user@example.com",password:" password "});expect(screen.getByRole("button",{name:"ログイン中..."})).toBeDisabled();expect(screen.getByLabelText("メールアドレス")).toBeDisabled();resolve({access_token:"access",token_type:"Bearer",expires_in:300,user:{id:"id",name:"Taro",email:"user@example.com",role:"user",status:"active"}});expect(await screen.findByText("TOP")).toBeInTheDocument();expect(localStorage.length).toBe(0)});
 it("clears password and shows API error with request id",async()=>{mockedLogin.mockRejectedValue(new ApiRequestError({status:401,code:"INVALID_CREDENTIALS",message:"認証失敗",request_id:"req-9"}));const user=userEvent.setup();renderPage();await user.type(screen.getByLabelText("メールアドレス"),"a@b.com");const password=screen.getByLabelText<HTMLInputElement>("パスワード");await user.type(password,"password");await user.click(screen.getByRole("button",{name:"ログイン"}));await waitFor(()=>expect(screen.getByText("認証失敗")).toBeInTheDocument());expect(screen.getByText("Request ID: req-9")).toBeInTheDocument();expect(password).toHaveValue("")});
});
