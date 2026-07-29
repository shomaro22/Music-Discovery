import type { RecommendationDetail, RecommendationType } from "../../types/recommendation";
import { RecommendationCard } from "./RecommendationCard";
const ORDER: Record<RecommendationType, number> = { YOUR_MATCH: 1, NEW_FAVORITE: 2, TRY_SOMETHING_NEW: 3 };
interface RecommendationListProps { recommendations: RecommendationDetail[]; }
export function RecommendationList({ recommendations }: RecommendationListProps) { return <div className="recommendation-list">{[...recommendations].sort((a,b) => ORDER[a.recommendation_type]-ORDER[b.recommendation_type]).map((detail) => <RecommendationCard key={detail.recommendation_detail_id} detail={detail} />)}</div>; }
