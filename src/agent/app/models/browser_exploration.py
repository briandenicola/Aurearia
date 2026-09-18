"""Strict, stateless DTOs for the Feature 360 exploration decision boundary."""

from typing import Annotated, Literal

from pydantic import BaseModel, ConfigDict, Field, StringConstraints, field_validator, model_validator

SCHEMA_VERSION = "aurearia.browser-exploration-decision/v1"

ActionName = Literal[
    "navigate",
    "click",
    "fill",
    "select",
    "upload_fixture",
    "set_viewport",
    "back",
    "wait_for_ui",
    "checkpoint",
    "finish",
]
EvidenceKind = Literal["route", "screenshot", "console", "network", "accessibility", "ui"]
Category = Literal[
    "functional",
    "console",
    "network",
    "accessibility",
    "visual",
    "navigation",
    "performance",
    "security",
    "other",
]
Severity = Literal["critical", "high", "medium", "low", "info"]

RunID = Annotated[str, StringConstraints(pattern=r"^aibr_[0-9]{8}T[0-9]{6}Z_[a-f0-9]{12}$")]
StepID = Annotated[str, StringConstraints(pattern=r"^step-[0-9]{3}$")]
EvidenceID = Annotated[str, StringConstraints(pattern=r"^ev_[a-z]+_[0-9]{4}$")]
Route = Annotated[str, StringConstraints(pattern=r"^/", max_length=200)]


class ExplorationModel(BaseModel):
    model_config = ConfigDict(extra="forbid", populate_by_name=True)


class Observation(ExplorationModel):
    evidence_id: EvidenceID = Field(alias="evidenceId")
    kind: EvidenceKind
    route: Route
    summary: Annotated[str, StringConstraints(max_length=2000)]


class DecisionRequest(ExplorationModel):
    schema_version: Literal["aurearia.browser-exploration-decision/v1"] = Field(alias="schemaVersion")
    run_id: RunID = Field(alias="runId")
    step_id: StepID = Field(alias="stepId")
    provider: Literal["anthropic", "ollama"]
    model: Annotated[str, StringConstraints(min_length=1, max_length=120)]
    workflow: Annotated[str, StringConstraints(min_length=1, max_length=80)]
    goal: Annotated[str, StringConstraints(min_length=1, max_length=1000)]
    allowed_actions: list[ActionName] = Field(alias="allowedActions", min_length=1)
    allowed_routes: list[Route] = Field(alias="allowedRoutes", min_length=1, max_length=30)
    observations: list[Observation] = Field(max_length=100)

    @field_validator("allowed_actions", "allowed_routes")
    @classmethod
    def values_must_be_unique(cls, values: list[str]) -> list[str]:
        if len(values) != len(set(values)):
            raise ValueError("values must be unique")
        return values


class ActionTarget(ExplorationModel):
    route: Route | None = None
    role: Annotated[str, StringConstraints(max_length=80)] | None = None
    accessible_name: Annotated[str, StringConstraints(max_length=200)] | None = Field(
        default=None, alias="accessibleName"
    )
    label: Annotated[str, StringConstraints(max_length=200)] | None = None
    value: Annotated[str, StringConstraints(max_length=1000)] | None = None
    fixture_id: Literal["synthetic-obverse-png", "synthetic-reverse-png"] | None = Field(
        default=None, alias="fixtureId"
    )
    width: int | None = Field(default=None, ge=320, le=1920)
    height: int | None = Field(default=None, ge=568, le=1080)
    milliseconds: int | None = Field(default=None, ge=0, le=5000)


class ModelUsage(ExplorationModel):
    input_tokens: int = Field(alias="inputTokens", ge=0)
    output_tokens: int = Field(alias="outputTokens", ge=0)


class ModelTriage(ExplorationModel):
    generated_by_model: Literal[True] = Field(alias="generatedByModel")
    summary: Annotated[str, StringConstraints(max_length=2000)]
    suggested_category: Category = Field(alias="suggestedCategory")
    suggested_severity: Severity = Field(alias="suggestedSeverity")
    confidence: float = Field(ge=0, le=1)
    evidence_ids: list[EvidenceID] = Field(alias="evidenceIds", min_length=1, max_length=20)

    @field_validator("evidence_ids")
    @classmethod
    def evidence_ids_must_be_unique(cls, values: list[str]) -> list[str]:
        if len(values) != len(set(values)):
            raise ValueError("evidenceIds must be unique")
        return values


class DecisionResponse(ExplorationModel):
    schema_version: Literal["aurearia.browser-exploration-decision/v1"] = Field(alias="schemaVersion")
    action: ActionName
    target: ActionTarget | None
    rationale: Annotated[str, StringConstraints(max_length=1000)]
    suspected_findings: list[ModelTriage] = Field(alias="suspectedFindings", max_length=10)
    usage: ModelUsage

    @model_validator(mode="after")
    def target_matches_action(self) -> "DecisionResponse":
        target = self.target
        if self.action == "navigate" and (target is None or target.route is None):
            raise ValueError("navigate requires target.route")
        if self.action in {"click", "fill", "select"}:
            if target is None or not any((target.role, target.accessible_name, target.label)):
                raise ValueError(f"{self.action} requires a bounded locator")
        if self.action in {"fill", "select"} and (target is None or target.value is None):
            raise ValueError(f"{self.action} requires target.value")
        if self.action == "upload_fixture" and (target is None or target.fixture_id is None):
            raise ValueError("upload_fixture requires target.fixtureId")
        if self.action == "set_viewport" and (
            target is None or target.width is None or target.height is None
        ):
            raise ValueError("set_viewport requires target.width and target.height")
        if self.action == "wait_for_ui" and (target is None or target.milliseconds is None):
            raise ValueError("wait_for_ui requires target.milliseconds")
        if self.action in {"back", "checkpoint", "finish"} and target is not None:
            raise ValueError(f"{self.action} requires a null target")
        return self


def validate_decision_for_request(request: DecisionRequest, response: DecisionResponse) -> DecisionResponse:
    """Validate response references against the one request; no state is retained."""

    if response.action not in request.allowed_actions:
        raise ValueError("decision action is not allowed by the request")
    if response.target is not None and response.target.route is not None:
        if response.target.route not in request.allowed_routes:
            raise ValueError("decision route is not allowed by the request")
    evidence_ids = {observation.evidence_id for observation in request.observations}
    referenced = {
        evidence_id
        for finding in response.suspected_findings
        for evidence_id in finding.evidence_ids
    }
    if not referenced.issubset(evidence_ids):
        raise ValueError("decision references evidence absent from the request")
    return response
