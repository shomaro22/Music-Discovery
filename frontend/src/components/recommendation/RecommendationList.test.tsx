import { expect, test } from "vitest";
import { render, screen } from "@testing-library/react";
import { RecommendationList } from "./RecommendationList";
import type { RecommendationDetail } from "../../types/recommendation";
const detail=(type:RecommendationDetail["recommendation_type"],name:string):RecommendationDetail=>({recommendation_detail_id:type,recommendation_type:type,song_id:type,song_name:name,artist_id:type,artist_name:"Artist",artwork_url:"",spotify_url:"https://open.spotify.com/track/test",reason:"理由",feedback_status:null});
test("推薦枠順で3曲を表示する",()=>{render(<RecommendationList recommendations={[detail("TRY_SOMETHING_NEW","Third"),detail("YOUR_MATCH","First"),detail("NEW_FAVORITE","Second")]}/>);expect(screen.getAllByRole("heading").map(v=>v.textContent)).toEqual(["First","Second","Third"]);});
