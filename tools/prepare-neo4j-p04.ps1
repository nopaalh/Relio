param([Parameter(Mandatory=$true)][string]$DatasetPath,[Parameter(Mandatory=$true)][string]$OutputDirectory)
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
$datasetDir=(Resolve-Path -LiteralPath $DatasetPath).Path
$targetDir=[IO.Path]::GetFullPath($OutputDirectory)
if ($targetDir.StartsWith($datasetDir+[IO.Path]::DirectorySeparatorChar,[StringComparison]::OrdinalIgnoreCase) -or $targetDir -eq $datasetDir) {throw 'Output must not overwrite dataset'}
$repoDir=Split-Path $PSScriptRoot -Parent
$expected=Get-Content -LiteralPath (Join-Path $repoDir 'backend/docs/handoff-o1-2026-10-09-8fd41a/source-cases-draft.json') -Raw -Encoding UTF8 | ConvertFrom-Json
$hashes=[ordered]@{}
$files=@('contact_employment_history.csv','crm_accounts.csv','crm_contacts.csv','crm_deals.csv','employees.csv','interactions.jsonl')
foreach($file in $files){$hash=(Get-FileHash -LiteralPath (Join-Path $datasetDir $file) -Algorithm SHA256).Hash.ToLowerInvariant();if($hash -ne $expected.expected_input_hashes.$file){throw "Checksum mismatch: $file"};$hashes[$file]=$hash}
function HashText([string]$s){$sha=[Security.Cryptography.SHA256]::Create();try{return ([BitConverter]::ToString($sha.ComputeHash([Text.Encoding]::UTF8.GetBytes($s)))).Replace('-','').ToLowerInvariant()}finally{$sha.Dispose()}}
function Json($value){ConvertTo-Json -InputObject $value -Depth 50 -Compress}
function Lit($value){Json $value}
function Fact($value,[string]$basis,[string[]]$ids){return [ordered]@{value=$value;state='known';temporal_basis=$basis;evidence_ids=@($ids);limitations=@()}}
$nodes=[Collections.Generic.List[object]]::new();$edges=[Collections.Generic.List[object]]::new();$events=[Collections.Generic.List[object]]::new();$evidence=[Collections.Generic.List[object]]::new();$deals=[Collections.Generic.List[object]]::new()
function Proof([string]$file,[string]$id,[string]$field,[string]$quote,[string]$date,[string]$basis,[string]$scope,[string[]]$eventIDs,[string[]]$dealIDs){
 $eid="ev:${file}:${id}:${field}"
 $evidence.Add([ordered]@{available_from=$date;value=[ordered]@{evidence_id=$eid;source_file=$file;source_record_id=$id;record_id_kind='native';source_key=[ordered]@{id=$id};source_checksum=$hashes[$file];source_field=$field;span=$null;source_date=$date;source_timestamp=$null;temporal_basis=$basis;content_excerpt=$quote;scope_kind=$scope;verification_state='verified';deal_ids=@($dealIDs);account_ids=@('P04');event_ids=@($eventIDs);edge_ids=@();occurrence_ids=@()}})
 return $eid
}
function Node([string]$id,[string]$type,[string]$native,$label,[string]$from,[string[]]$proofs,[string[]]$eventIDs,$kind){
 $nodes.Add([ordered]@{available_from=$from;value=[ordered]@{node_id=$id;node_type=$type;entity_id=$native;person_kind=$kind;person=$null;label=$label;validity=[ordered]@{valid_from=$from;valid_to=$null;open_end_reason='source_does_not_supply_end';temporal_basis='event'};verification_state='verified';evidence_ids=@($proofs);event_ids=@($eventIDs)}})
}
function Edge([string]$type,[string]$source,[string]$target,[string]$from,[string[]]$proofs,[string[]]$eventIDs){
 $id='edge:'+(HashText (Json @($type,$source,$target,$proofs,$eventIDs,'relio-neo4j-p04-v1')))
 $edges.Add([ordered]@{available_from=$from;value=[ordered]@{edge_id=$id;edge_type=$type;source=$source;target=$target;validity=[ordered]@{valid_from=$from;valid_to=$null;open_end_reason='source_does_not_supply_end';temporal_basis='event'};source_timestamp=$null;observed_at=$null;recorded_at=$null;verification_state='verified';match_method='native_reference';evidence_ids=@($proofs);event_ids=@($eventIDs)}})
 foreach($p in $proofs){foreach($e in $evidence){if($e.value.evidence_id -eq $p){$e.value.edge_ids=@($e.value.edge_ids)+$id}}}
 return $id
}
function Participant($id,$raw,[string]$state,[string]$method,[string[]]$proofs){return [ordered]@{node_id=$id;raw_ref=$raw;verification_state=$state;match_method=$method;candidate_node_ids=@();evidence_ids=@($proofs)}}
$d=@(Import-Csv -LiteralPath (Join-Path $datasetDir 'crm_deals.csv') -Encoding UTF8 | Where-Object deal_id -eq 'DL-004');if($d.Count -ne 1 -or $d[0].account_id -ne 'P04'){throw 'Invalid P04 deal'};$d=$d[0]
$a=@(Import-Csv -LiteralPath (Join-Path $datasetDir 'crm_accounts.csv') -Encoding UTF8 | Where-Object account_id -eq 'P04');if($a.Count -ne 1){throw 'Invalid P04 account'};$a=$a[0]
$contacts=@(Import-Csv -LiteralPath (Join-Path $datasetDir 'crm_contacts.csv') -Encoding UTF8)
$employees=@(Import-Csv -LiteralPath (Join-Path $datasetDir 'employees.csv') -Encoding UTF8)
$interactions=@(Get-Content -LiteralPath (Join-Path $datasetDir 'interactions.jsonl') -Encoding UTF8 | ForEach-Object{ConvertFrom-Json $_} | Where-Object account_id -eq 'P04' | Sort-Object tanggal,interaction_id)
if($interactions.Count -ne 3 -or ($interactions.interaction_id -join ',') -ne 'I0284,I0314,I0335'){throw 'Unexpected P04 interaction coverage'}
$identity=Proof 'crm_deals.csv' $d.deal_id 'account_id' $d.account_id $d.dibuat 'event' 'deal' @() @('DL-004')
$created=Proof 'crm_deals.csv' $d.deal_id 'dibuat' $d.dibuat $d.dibuat 'event' 'deal' @() @('DL-004')
$dealValue=[ordered]@{deal_id=$d.deal_id;account_id=$d.account_id;deal_type=$null;owner_id=$null;stage=$null;stage_since=$null;created_at=(Fact $d.dibuat 'event' @($created));planned_outlets=$null;potential_acv_idr=$null;status=$null;record_evidence_ids=@($identity,$created)}
foreach($pair in @(@('deal_type','tipe'),@('owner_id','owner_id'),@('stage','stage'),@('stage_since','stage_sejak'),@('planned_outlets','outlet'),@('potential_acv_idr','nilai_tahunan'),@('status','status'))){
 $key=$pair[0];$field=$pair[1];$p=Proof 'crm_deals.csv' $d.deal_id $field $d.$field '2026-10-01' 'snapshot' 'deal' @() @('DL-004');$value=$d.$field
 if($key -eq 'planned_outlets'){$value=[int]$value};if($key -eq 'potential_acv_idr'){$value=[long]$value};$dealValue[$key]=Fact $value 'snapshot' @($p);$dealValue.record_evidence_ids+=@($p)
}
$deals.Add([ordered]@{available_from=$d.dibuat;value=$dealValue})
Node 'deal:DL-004' 'deal' 'DL-004' (Fact 'DL-004' 'event' @($identity)) $d.dibuat @($identity) @() $null
Node 'account:P04' 'account' 'P04' (Fact 'P04' 'event' @($identity)) $d.dibuat @($identity) @() $null
$null=Edge 'ACCOUNT_CONTEXT' 'deal:DL-004' 'account:P04' $d.dibuat @($identity) @()
$people=@{}
foreach($i in $interactions){
 [datetime]::ParseExact($i.tanggal,'yyyy-MM-dd',[Globalization.CultureInfo]::InvariantCulture) | Out-Null
 $eventID='event:interaction:'+$i.interaction_id
 $body=Proof 'interactions.jsonl' $i.interaction_id 'isi' $i.isi $i.tanggal 'event' 'account' @($eventID) @()
 $subject=Proof 'interactions.jsonl' $i.interaction_id 'subjek' $i.subjek $i.tanggal 'event' 'account' @($eventID) @()
 $link=Proof 'interactions.jsonl' $i.interaction_id 'account_id' $i.account_id $i.tanggal 'event' 'account' @($eventID) @()
 $typeProof=Proof 'interactions.jsonl' $i.interaction_id 'tipe' $i.tipe $i.tanggal 'event' 'account' @($eventID) @()
 Node $eventID 'interaction' $i.interaction_id (Fact $i.subjek 'event' @($subject)) $i.tanggal @($subject,$body,$link,$typeProof) @($eventID) $null
 $edgeIDs=@((Edge 'CONCERNS_ACCOUNT' $eventID 'account:P04' $i.tanggal @($link) @($eventID)))
 $participants=@();$actor=@();$targets=@();$nodeIDs=@($eventID,'account:P04')
 if($i.peserta){
  $p=Proof 'interactions.jsonl' $i.interaction_id 'peserta' $i.peserta $i.tanggal 'event' 'account' @($eventID) @()
  foreach($native in ($i.peserta -split ';')){
   $kind=if($native.StartsWith('K')){'contact'}else{'employee'};$personNodeID="${kind}:${native}"
   $rows=@(if($kind -eq 'contact'){$contacts | Where-Object contact_id -eq $native}else{$employees | Where-Object employee_id -eq $native});if($rows.Count -ne 1){throw "Invalid participant $native"}
   if(!$people.ContainsKey($personNodeID)){Node $personNodeID 'person' $native (Fact $native 'event' @($p)) $i.tanggal @($p) @($eventID) $kind;$people[$personNodeID]=$true}
   $participants+=@(Participant $personNodeID $native 'verified' 'native_meeting_id' @($p));$nodeIDs+=@($personNodeID);$edgeIDs+=@((Edge 'HAS_PARTICIPANT' $eventID $personNodeID $i.tanggal @($p) @($eventID)))
  }
 }
 foreach($role in @('dari','ke')){if($i.$role){$p=Proof 'interactions.jsonl' $i.interaction_id $role $i.$role $i.tanggal 'event' 'account' @($eventID) @();$ref=Participant $null $i.$role 'unresolved' 'raw_address_no_historical_alias' @($p);if($role -eq 'dari'){$actor+=@($ref)}else{$targets+=@($ref)}}}
 $eventProofs=@($evidence | Where-Object {$_.value.source_record_id -eq $i.interaction_id} | ForEach-Object{$_.value.evidence_id})
 $events.Add([ordered]@{event_id=$eventID;verification_state='verified';event_at=$i.tanggal;time_precision='date';source_timestamp=$null;event_type=$i.tipe;status=[ordered]@{value=$null;state='unknown';temporal_basis='event';evidence_ids=@();limitations=@('source_has_no_execution_status')};raw_status=$null;summary=(Fact $i.isi 'event' @($body));actors=@($actor);targets=@($targets);participants=@($participants);scope_kind='account';deal_ids=@();account_ids=@('P04');node_ids=@($nodeIDs);edge_ids=@($edgeIDs);evidence_ids=@($eventProofs)})
}
$schemaVersion='relio-neo4j-p04-v1';$version='p04:'+(HashText (Json @($schemaVersion,$hashes)))
$counts=[ordered]@{deals=$deals.Count;nodes=$nodes.Count;edges=$edges.Count;events=$events.Count;evidence=$evidence.Count}
$payload=[ordered]@{deals=@($deals.ToArray());nodes=@($nodes.ToArray());edges=@($edges.ToArray());events=@($events.ToArray());evidence=@($evidence.ToArray())}
$registry=[Collections.Generic.List[object]]::new()
foreach($recordKind in @('deals','nodes','edges','events','evidence')){foreach($record in $payload[$recordKind]){$value=if($recordKind -eq 'events'){$record}else{$record.value};$idField=@{deals='deal_id';nodes='node_id';edges='edge_id';events='event_id';evidence='evidence_id'}[$recordKind];$registry.Add([ordered]@{kind=$recordKind;id=$value[$idField];hash=(HashText (Json $record))})}}
$sortedRegistry=@($registry | ForEach-Object {[pscustomobject]$_} | Sort-Object kind,id)
$sealLines=[Collections.Generic.List[string]]::new();$sealLines.Add('schema:'+$schemaVersion);$sealLines.Add('version:'+$version)
foreach($key in @($hashes.Keys | Sort-Object)){$sealLines.Add("source:${key}:$($hashes[$key])")};foreach($key in @($counts.Keys | Sort-Object)){$sealLines.Add("count:${key}:$($counts[$key])")}
foreach($record in $sortedRegistry){$sealLines.Add("record:$($record.kind):$($record.id):$($record.hash)")};$sealLines.Add('deal:DL-004');$sealLines.Add('account:P04')
$manifestHash=HashText ($sealLines -join "`n")
$manifest=[ordered]@{dataset_version=$version;schema_version=$schemaVersion;manifest_hash=$manifestHash;ready=$false;deal_ids=@('DL-004');account_ids=@('P04');sources=$hashes;counts=$counts;records=$sortedRegistry}
$context=[ordered]@{manifest=$manifest;deals=$payload.deals;nodes=$payload.nodes;edges=$payload.edges;events=$payload.events;evidence=$payload.evidence}
$schema=@'
CREATE CONSTRAINT relio_dataset IF NOT EXISTS FOR (d:RelioDataset) REQUIRE d.dataset_version IS UNIQUE;
CREATE CONSTRAINT relio_record IF NOT EXISTS FOR (n:RelioRecord) REQUIRE (n.dataset_version,n.record_kind,n.record_id) IS UNIQUE;
'@
$seed=[Collections.Generic.List[string]]::new()
$v=Lit $version;$h=Lit $manifestHash
$seed.Add("MERGE (d:RelioDataset {dataset_version:$v}) ON CREATE SET d.manifest_hash=$h, d.schema_version=$(Lit $schemaVersion), d.payload=$(Lit (Json $manifest)), d.ready=false;")
$expectedRows=[Collections.Generic.List[object]]::new()
foreach($kind in @('deals','nodes','edges','events','evidence')){
 foreach($row in $payload[$kind]){
  $value=if($kind -eq 'events'){$row}else{$row.value};$idField=@{deals='deal_id';nodes='node_id';edges='edge_id';events='event_id';evidence='evidence_id'}[$kind];$id=$value[$idField];$from=if($kind -eq 'events'){$row.event_at}else{$row.available_from};$json=Json $row;$digest=HashText $json
  $expectedRows.Add([ordered]@{kind=$kind;id=$id;hash=$digest;payload=$json;from=$from;source=$(if($kind -eq 'edges'){$value.source}else{''});target=$(if($kind -eq 'edges'){$value.target}else{''});type=$(if($kind -eq 'edges'){$value.edge_type}else{''})})
  $extra=if($kind -eq 'edges'){", n.source=$(Lit $value.source), n.target=$(Lit $value.target), n.edge_type=$(Lit $value.edge_type)"}else{''}
  $seed.Add("MATCH (d:RelioDataset {dataset_version:$v}) WHERE d.manifest_hash=$h MERGE (n:RelioRecord {dataset_version:$v,record_kind:$(Lit $kind),record_id:$(Lit $id)}) ON CREATE SET n.payload=$(Lit $json), n.payload_hash=$(Lit $digest), n.available_from=$(Lit $from)$extra;")
 }
}
# Adjacency is real; the generic CONTEXT_LINK stores the domain edge_type.
$seed.Add("MATCH (d:RelioDataset {dataset_version:$v}) WHERE d.manifest_hash=$h MATCH (e:RelioRecord {dataset_version:$v,record_kind:'edges'}) MATCH (a:RelioRecord {dataset_version:$v,record_kind:'nodes',record_id:e.source}), (b:RelioRecord {dataset_version:$v,record_kind:'nodes',record_id:e.target}) MERGE (a)-[r:CONTEXT_LINK {dataset_version:$v,edge_id:e.record_id}]->(b) ON CREATE SET r.edge_type=e.edge_type;")
$checks=Lit @($expectedRows.ToArray())
$expectedLiteral=(@($expectedRows | ForEach-Object{"{kind:$(Lit $_.kind),id:$(Lit $_.id),hash:$(Lit $_.hash),payload:$(Lit $_.payload),from:$(Lit $_.from),source:$(Lit $_.source),target:$(Lit $_.target),type:$(Lit $_.type)}"}) -join ',')
$expectedEdges=(@($edges | ForEach-Object{"{id:$(Lit $_.value.edge_id),source:$(Lit $_.value.source),target:$(Lit $_.value.target),type:$(Lit $_.value.edge_type)}"}) -join ',')
$manifestLiteral=Lit (Json $manifest)
$seed.Add("MATCH (d:RelioDataset {dataset_version:$v}) WHERE d.manifest_hash=$h WITH d, [$expectedLiteral] AS expected UNWIND expected AS x OPTIONAL MATCH (n:RelioRecord {dataset_version:$v,record_kind:x.kind,record_id:x.id}) WITH d, count(x) AS expected_count, sum(CASE WHEN n.payload_hash=x.hash AND n.payload=x.payload AND n.available_from=x.from AND (x.kind <> 'edges' OR (n.source=x.source AND n.target=x.target AND n.edge_type=x.type)) THEN 1 ELSE 0 END) AS correct MATCH (allRecords:RelioRecord {dataset_version:$v}) WITH d,expected_count,correct,count(allRecords) AS actual OPTIONAL MATCH (a)-[r:CONTEXT_LINK {dataset_version:$v}]->(b) WITH d,expected_count,correct,actual,collect(CASE WHEN r IS NOT NULL THEN {id:r.edge_id,source:a.record_id,target:b.record_id,type:r.edge_type,source_kind:a.record_kind,target_kind:b.record_kind,source_version:a.dataset_version,target_version:b.dataset_version} ELSE null END) AS links SET d.ready=(d.payload=$manifestLiteral AND d.schema_version=$(Lit $schemaVersion) AND correct=expected_count AND actual=expected_count AND size(links)=$($edges.Count) AND all(x IN [$expectedEdges] WHERE single(r IN links WHERE r.id=x.id AND r.source=x.source AND r.target=x.target AND r.type=x.type AND r.source_kind='nodes' AND r.target_kind='nodes' AND r.source_version=$v AND r.target_version=$v))) RETURN d.dataset_version AS dataset_version,d.ready AS ready,actual,size(links) AS links;")
if(!(Test-Path -LiteralPath $targetDir)){New-Item -ItemType Directory -Path $targetDir | Out-Null}
$utf8=[Text.UTF8Encoding]::new($false)
foreach($entry in @(@('context-p04.json',(Json $context)),@('manifest-p04.json',(Json $manifest)),@('schema.cypher',$schema),@('seed-p04.cypher',($seed -join "`n")))){[IO.File]::WriteAllText((Join-Path $targetDir $entry[0]),($entry[1].Replace("`r`n","`n")+"`n"),$utf8)}
Write-Output "Prepared $version (ready=false until explicit Aura load/verification)."
